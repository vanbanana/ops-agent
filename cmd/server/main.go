package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"ops-agent/internal/agent"
	"ops-agent/internal/api"
	"ops-agent/internal/audit"
	"ops-agent/internal/config"
	"ops-agent/internal/llm"
	mcpkg "ops-agent/internal/mcp"
	"ops-agent/internal/permission"
	"ops-agent/internal/safety"
	"ops-agent/internal/store"
	"ops-agent/internal/tools"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Build-time variables (set via -ldflags)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Load .env
	config.LoadDotEnv(".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	// Initialize LLM client (hot-reloadable)
	llmClient := agent.NewHotReloadClient(agent.ClientConfig{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
		Timeout: 60 * time.Second,
	})

	// Initialize tool registry with all probes
	registry := tools.NewRegistry()
	tools.RegisterAllProbes(registry)
	tools.RegisterWriteTools(registry)

	// Initialize MCP Manager (connect to external MCP Servers)
	mcpManager := mcpkg.NewManager(registry)

	// Initialize SQLite database (used for audit logs + session persistence)
	db, err := store.OpenDB(cfg.DBPath)
	if err != nil {
		log.Printf("warning: SQLite unavailable (%v), audit logs disabled", err)
	} else {
		defer db.Close()
	}

	// Initialize model pool (load from DB > providers.json > .env fallback)
	modelPool := llm.NewModelPool(db, "providers.json", cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)

	// Load and connect MCP Servers from DB
	if db != nil {
		var mcpConfigs []mcpkg.ServerConfig
		rows, err := db.Query(`SELECT id, name, transport, command, args, url, env, is_active FROM mcp_servers`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var s mcpkg.ServerConfig
				var argsJSON, envJSON string
				var isActive int
				if err := rows.Scan(&s.ID, &s.Name, &s.Transport, &s.Command, &argsJSON, &s.URL, &envJSON, &isActive); err != nil {
					continue
				}
				json.Unmarshal([]byte(argsJSON), &s.Args)
				json.Unmarshal([]byte(envJSON), &s.Env)
				s.IsActive = isActive == 1
				mcpConfigs = append(mcpConfigs, s)
			}
		}
		// If no MCP servers configured, seed with Context7 (remote SSE, no local deps)
		if len(mcpConfigs) == 0 {
			context7 := mcpkg.ServerConfig{
				ID: "context7", Name: "context7", Transport: "streamable",
				URL:      "https://mcp.context7.com/mcp",
				IsActive: true,
			}
			mcpConfigs = append(mcpConfigs, context7)
			// Persist to DB
			argsJSON, _ := json.Marshal(context7.Args)
			envJSON, _ := json.Marshal(context7.Env)
			db.Exec(`INSERT OR IGNORE INTO mcp_servers (id, name, transport, command, args, url, env, is_active) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				context7.ID, context7.Name, context7.Transport, context7.Command, string(argsJSON), context7.URL, string(envJSON), 1)
		}
		mcpManager.ConnectAll(mcpConfigs)
	}

	modelPool.SetOnChange(func(active llm.Provider) {
		llmClient.Reload(agent.ClientConfig{
			BaseURL: active.BaseURL,
			APIKey:  active.APIKey,
			Model:   active.ModelID,
			Timeout: 60 * time.Second,
		})
		log.Printf("model switched to: %s (%s)", active.Name, active.ModelID)
	})

	// If pool has an active provider different from env, apply it now
	if active, ok := modelPool.GetActive(); ok {
		llmClient.Reload(agent.ClientConfig{
			BaseURL: active.BaseURL,
			APIKey:  active.APIKey,
			Model:   active.ModelID,
			Timeout: 60 * time.Second,
		})
	}

	// Initialize session store (SQLite persistent)
	var sessions store.SessionStoreInterface
	if db != nil {
		sessions = store.NewSQLiteSessionStore(db)
	} else {
		sessions = store.NewSessionStore() // fallback to in-memory if DB unavailable
	}

	// Initialize audit writer
	auditWriter := audit.NewWriter(db) // nil-safe: noop if db is nil

	// Initialize permission service
	permSvc := permission.NewService()

	// Initialize agent
	agentInstance := agent.NewAgent(llmClient, registry, agent.AgentConfig{
		Model: cfg.LLMModel,
	}, permSvc)
	// Register multi-agent tool (OpenCode pattern: model decides when to use it)
	tools.RegisterMultiAgentTool(registry, agentInstance.MultiAgent())
	// Wire audit writer into agent loop
	agentInstance.SetAuditWriter(&auditBridge{w: auditWriter})

	// Initialize JWT auth
	authService := api.NewAuthService(api.AuthConfig{
		Secret:   cfg.JWTSecret,
		Username: "admin",
		Password: getEnvOrDefault("ADMIN_PASSWORD", "admin123"),
	})

	// Setup router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(120 * time.Second))

	// CORS middleware for production deployment
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// Rate limiting: 200 req/min per token/IP
	rateLimiter := api.NewRateLimiter(200, 1*time.Minute)
	r.Use(rateLimiter.Middleware)

	// Auth middleware: enforce JWT only in production (non-default secret)
	isDevMode := cfg.JWTSecret == "dev-only-insecure-key" || cfg.JWTSecret == "dev-only-not-for-production" || strings.HasPrefix(cfg.JWTSecret, "dev-")
	if !isDevMode {
		// Protected: all /api/v1/* except login require JWT
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				path := req.URL.Path
				// Skip auth for: login, health, version, static assets
				if path == "/api/v1/auth/login" || path == "/health" || path == "/health/deep" || path == "/version" {
					next.ServeHTTP(w, req)
					return
				}
				// Apply JWT middleware for all other API routes
				if strings.HasPrefix(path, "/api/") {
					authService.JWTMiddleware(next).ServeHTTP(w, req)
					return
				}
				next.ServeHTTP(w, req)
			})
		})
		fmt.Println("   Auth: JWT enforced (production mode)")
	} else {
		fmt.Println("   Auth: DISABLED (dev mode, set JWT_SECRET to enable)")
	}

	// Public routes (no auth)
	r.Post("/api/v1/auth/login", authService.HandleLogin)
	r.Get("/api/v1/auth/lockouts", authService.HandleListLockouts)
	r.Delete("/api/v1/auth/lockout/{ip}", authService.HandleUnlockIP)

	// Health check — quick (no LLM ping, for frontend polling)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		activeModel := cfg.LLMModel
		if active, ok := modelPool.GetActive(); ok {
			activeModel = active.ModelID
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "healthy",
			"components": map[string]any{
				"llm": map[string]any{"vendor": activeModel, "status": "configured"},
			},
		})
	})

	// Deep health check — pings LLM (10.12: returns degraded when unreachable)
	r.Get("/health/deep", func(w http.ResponseWriter, r *http.Request) {
		activeModel := cfg.LLMModel
		if active, ok := modelPool.GetActive(); ok {
			activeModel = active.ModelID
		}
		llmStatus := "up"
		overallStatus := "healthy"

		pingCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		_, err := llmClient.Chat(pingCtx, agent.ChatRequest{
			Messages: []agent.Message{{Role: "user", Content: "ping"}},
		})
		if err != nil {
			llmStatus = "down"
			overallStatus = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": overallStatus,
			"components": map[string]any{
				"llm": map[string]any{"vendor": activeModel, "status": llmStatus},
			},
		})
	})

	// Version info
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"version":    Version,
			"git_commit": GitCommit,
			"build_time": BuildTime,
			"go_version": runtime.Version(),
			"platform":   runtime.GOOS + "/" + runtime.GOARCH,
		})
	})

	// Tools list
	r.Get("/api/v1/tools", func(w http.ResponseWriter, r *http.Request) {
		defs := registry.Definitions()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"tools": defs,
				"count": len(defs),
			},
		})
	})

	// Tools status (all tools with enable/disable state — for settings UI)
	r.Get("/api/v1/tools/status", func(w http.ResponseWriter, r *http.Request) {
		status := registry.AllStatus()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": status})
	})

	// Toggle tool enable/disable
	r.Post("/api/v1/tools/toggle", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name    string `json:"name"`
			Enabled bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request"})
			return
		}
		if req.Name == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "name is required"})
			return
		}
		if _, ok := registry.Get(req.Name); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"code": 404, "error": "tool not found"})
			return
		}
		if req.Enabled {
			registry.Enable(req.Name)
		} else {
			registry.Disable(req.Name)
		}
		// Persist to configs table
		if db != nil {
			val := "1"
			if !req.Enabled {
				val = "0"
			}
			db.Exec(`INSERT OR REPLACE INTO configs (key, value, updated_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
				"tool_enabled_"+req.Name, val)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"name": req.Name, "enabled": req.Enabled}})
	})

	// Chat stream (SSE)
	r.Post("/api/v1/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			Message   string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"code":400,"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, `{"code":400,"error":"message is required"}`, http.StatusBadRequest)
			return
		}
		if req.SessionID == "" {
			req.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
		}

		// Get or create session, load history
		sessions.GetOrCreate(req.SessionID)
		history := sessionsToAgentMessages(sessions.GetRecentMessages(req.SessionID, cfg.MaxHistoryMessages))

		// Store user message
		sessions.AppendMessage(req.SessionID, store.Message{Role: "user", Content: req.Message})

		// SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		events := agentInstance.RunStream(ctx, req.SessionID, req.Message, history)

		// Task 9.5: heartbeat goroutine (20s ping to keep connection alive)
		done := make(chan struct{})
		go func() {
			ticker := time.NewTicker(20 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case t := <-ticker.C:
					fmt.Fprintf(w, "event: ping\ndata: {\"ts\":\"%s\"}\n\n", t.Format(time.RFC3339))
					flusher.Flush()
				}
			}
		}()

		for event := range events {
			data, _ := json.Marshal(event.Data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
			flusher.Flush()

			// Store assistant reply in session
			if event.Type == "output" {
				if reply, ok := event.Data["reply"].(string); ok {
					sessions.AppendMessage(req.SessionID, store.Message{Role: "assistant", Content: reply})
				}
			}
		}
		close(done)
	})

	// Sync chat (non-streaming)
	r.Post("/api/v1/chat", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			Message   string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"code":400,"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Message == "" {
			http.Error(w, `{"code":400,"error":"message is required"}`, http.StatusBadRequest)
			return
		}
		if req.SessionID == "" {
			req.SessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
		}

		sessions.GetOrCreate(req.SessionID)
		history := sessionsToAgentMessages(sessions.GetRecentMessages(req.SessionID, cfg.MaxHistoryMessages))
		sessions.AppendMessage(req.SessionID, store.Message{Role: "user", Content: req.Message})

		ctx := r.Context()
		events := agentInstance.RunStream(ctx, req.SessionID, req.Message, history)

		var reply string
		var lastEvent agent.Event
		for event := range events {
			lastEvent = event
			if event.Type == "output" {
				if r, ok := event.Data["reply"].(string); ok {
					reply = r
				}
			}
		}

		if reply != "" {
			sessions.AppendMessage(req.SessionID, store.Message{Role: "assistant", Content: reply})
		}

		w.Header().Set("Content-Type", "application/json")
		status := "ok"
		if lastEvent.Type == "done" {
			if s, ok := lastEvent.Data["status"].(string); ok {
				status = s
			}
		}

		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"session_id": req.SessionID,
				"reply":      reply,
				"status":     status,
			},
		})
	})

	// Safety scan (test single command)
	r.Get("/api/v1/safety/scan", func(w http.ResponseWriter, r *http.Request) {
		cmd := r.URL.Query().Get("cmd")
		if cmd == "" {
			http.Error(w, `{"code":400,"error":"cmd parameter required"}`, http.StatusBadRequest)
			return
		}
		result := safety.ValidateCommand(cmd)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"command": cmd,
				"status":  string(result.Status),
				"reason":  string(result.Reason),
				"detail":  result.Detail,
				"rule_id": result.RuleID,
			},
		})
	})

	// File system API (Task 15 — desktop file manager)
	r.Get("/api/v1/fs/list", api.HandleFSList)

	// Sessions API (persistent)
	r.Get("/api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		list := sessions.ListSessions()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": list})
	})

	r.Delete("/api/v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := sessions.DeleteSession(id); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{"code": 500, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "ok"}})
	})

	r.Get("/api/v1/sessions/{id}/messages", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		msgs := sessions.GetRecentMessages(id, 100)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": msgs})
	})

	// Session detail (metadata + messages)
	r.Get("/api/v1/sessions/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		msgs := sessions.GetRecentMessages(id, 100)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"id":       id,
				"messages": msgs,
			},
		})
	})
	r.Get("/api/v1/fs/stat", api.HandleFSStat)
	r.Post("/api/v1/fs/mkdir", api.HandleFSMkdir)
	r.Post("/api/v1/fs/rename", api.HandleFSRename)
	r.Post("/api/v1/fs/copy", api.HandleFSCopy)
	r.Post("/api/v1/fs/move", api.HandleFSMove)
	r.Post("/api/v1/fs/delete", api.HandleFSDelete)

	// Permission respond endpoint
	r.Post("/api/v1/permission/respond", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RequestID string `json:"request_id"`
			Action    string `json:"action"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request body"})
			return
		}
		if req.RequestID == "" || req.Action == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "request_id and action are required"})
			return
		}

		if err := permSvc.Respond(req.RequestID, req.Action); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"code": 404, "error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "ok"}})
	})

	// Permission mode endpoint
	r.Get("/api/v1/permission/mode", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"mode": string(permSvc.GetMode())}})
	})

	r.Put("/api/v1/permission/mode", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request body"})
			return
		}
		switch permission.Mode(req.Mode) {
		case permission.ModeDefault, permission.ModeAutoApprove, permission.ModePlan:
			permSvc.SetMode(permission.Mode(req.Mode))
			// Activate/deactivate plan state machine accordingly
			if permission.Mode(req.Mode) == permission.ModePlan {
				agentInstance.PlanState().StartPlanning(fmt.Sprintf("plan_%d", time.Now().UnixNano()))
			} else {
				agentInstance.PlanState().Reset()
			}
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "mode must be 'default', 'auto_approve', or 'plan'"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"mode": req.Mode}})
	})

	// Plan mode approve/reject
	r.Post("/api/v1/plan/approve", func(w http.ResponseWriter, r *http.Request) {
		agentInstance.PlanState().Approve()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "approved"}})
	})

	r.Post("/api/v1/plan/reject", func(w http.ResponseWriter, r *http.Request) {
		agentInstance.PlanState().Reject()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "rejected"}})
	})

	// Risk preview (Task 12)
	previewEngine := safety.NewPreviewEngine()

	// Configs CRUD (for whitelist and other settings)
	r.Get("/api/v1/configs", func(w http.ResponseWriter, r *http.Request) {
		keys := r.URL.Query().Get("keys")
		result := map[string]string{}
		if db != nil && keys != "" {
			for _, k := range strings.Split(keys, ",") {
				k = strings.TrimSpace(k)
				var val string
				err := db.QueryRow("SELECT value FROM configs WHERE key = ?", k).Scan(&val)
				if err == nil {
					result[k] = val
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": result})
	})

	r.Put("/api/v1/configs", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request body"})
			return
		}
		if db == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]any{"code": 503, "error": "database not available"})
			return
		}
		for k, v := range body {
			val := fmt.Sprintf("%v", v)
			if s, ok := v.(string); ok {
				val = s
			}
			_, err := db.Exec(`INSERT OR REPLACE INTO configs (key, value, updated_at) VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'))`, k, val)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{"code": 500, "error": err.Error()})
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "ok"}})
	})

	// Model pool API
	modelsHandler := &api.ModelsHandler{Pool: modelPool}
	r.Get("/api/v1/models/pool", modelsHandler.HandleGetPool)
	r.Put("/api/v1/models/pool", modelsHandler.HandleSavePool)
	r.Post("/api/v1/models/switch", modelsHandler.HandleSwitch)
	r.Post("/api/v1/models/test", modelsHandler.HandleTest)

	// MCP Servers management API
	r.Get("/api/v1/mcp/servers", func(w http.ResponseWriter, r *http.Request) {
		servers := mcpManager.GetServers()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": servers})
	})

	r.Post("/api/v1/mcp/servers", func(w http.ResponseWriter, r *http.Request) {
		var srv mcpkg.ServerConfig
		if err := json.NewDecoder(r.Body).Decode(&srv); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request"})
			return
		}
		if srv.ID == "" || srv.Name == "" || srv.Transport == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "id, name, transport required"})
			return
		}
		// Persist to DB
		if db != nil {
			argsJSON, _ := json.Marshal(srv.Args)
			envJSON, _ := json.Marshal(srv.Env)
			isActive := 0
			if srv.IsActive {
				isActive = 1
			}
			db.Exec(`INSERT OR REPLACE INTO mcp_servers (id, name, transport, command, args, url, env, is_active) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				srv.ID, srv.Name, srv.Transport, srv.Command, string(argsJSON), srv.URL, string(envJSON), isActive)
		}
		// Reconnect if active
		if srv.IsActive {
			go mcpManager.ConnectAll([]mcpkg.ServerConfig{srv})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": srv})
	})

	r.Delete("/api/v1/mcp/servers/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if db != nil {
			db.Exec(`DELETE FROM mcp_servers WHERE id = ?`, id)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "ok"}})
	})

	// Terminal exec API
	r.Post("/api/v1/terminal/exec", api.HandleTerminalExec)

	// Audit log read API
	r.Get("/api/v1/audit/logs", func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []any{}})
			return
		}
		limit := 100
		rows, err := db.Query(`SELECT id, trace_id, session_id, round_number, stage, role, content, triggered_by, status, duration_ms, created_at FROM audit_logs ORDER BY created_at DESC LIMIT ?`, limit)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []any{}})
			return
		}
		defer rows.Close()
		var logs []map[string]any
		for rows.Next() {
			var id int
			var traceID, sessionID, stage, content, triggeredBy, status, createdAt string
			var role *string
			var roundNumber, durationMs int
			if err := rows.Scan(&id, &traceID, &sessionID, &roundNumber, &stage, &role, &content, &triggeredBy, &status, &durationMs, &createdAt); err != nil {
				continue
			}
			entry := map[string]any{
				"id": id, "trace_id": traceID, "session_id": sessionID,
				"round": roundNumber, "stage": stage, "content": content,
				"triggered_by": triggeredBy, "status": status,
				"duration_ms": durationMs, "created_at": createdAt,
			}
			if role != nil {
				entry["role"] = *role
			}
			logs = append(logs, entry)
		}
		if logs == nil {
			logs = []map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": logs})
	})

	// Audit trace — single request full chain by trace_id
	r.Get("/api/v1/audit/{traceID}", func(w http.ResponseWriter, r *http.Request) {
		traceID := chi.URLParam(r, "traceID")
		if db == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []any{}})
			return
		}
		rows, err := db.Query(`SELECT id, trace_id, session_id, round_number, stage, role, content, triggered_by, status, duration_ms, created_at FROM audit_logs WHERE trace_id = ? ORDER BY created_at ASC`, traceID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": []any{}})
			return
		}
		defer rows.Close()
		var entries []map[string]any
		for rows.Next() {
			var id int
			var tid, sessionID, stage, content, triggeredBy, status, createdAt string
			var role *string
			var roundNumber, durationMs int
			if err := rows.Scan(&id, &tid, &sessionID, &roundNumber, &stage, &role, &content, &triggeredBy, &status, &durationMs, &createdAt); err != nil {
				continue
			}
			entry := map[string]any{
				"id": id, "trace_id": tid, "session_id": sessionID,
				"round": roundNumber, "stage": stage, "content": content,
				"triggered_by": triggeredBy, "status": status,
				"duration_ms": durationMs, "created_at": createdAt,
			}
			if role != nil {
				entry["role"] = *role
			}
			entries = append(entries, entry)
		}
		if entries == nil {
			entries = []map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": entries})
	})

	// Active model info (lightweight, for frontend header display)
	r.Get("/api/v1/models/active", func(w http.ResponseWriter, r *http.Request) {
		active, ok := modelPool.GetActivePublic()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"model": cfg.LLMModel}})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": active})
	})

	// Desktop probe — direct tool call without LLM (Task 14)
	r.Post("/api/v1/desktop/probe/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		probeName := "probe_" + name
		if _, ok := registry.Get(probeName); !ok {
			// Try without prefix
			probeName = name
			if _, ok := registry.Get(probeName); !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]any{
					"code": 404, "error": fmt.Sprintf("probe %q not found", name),
				})
				return
			}
		}

		var args map[string]any
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
				args = map[string]any{}
			}
		}
		if args == nil {
			args = map[string]any{}
		}

		result := registry.Dispatch(r.Context(), probeName, args)
		w.Header().Set("Content-Type", "application/json")
		if result.Error != "" {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"code":  500,
				"error": result.Error,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"tool":    probeName,
				"result":  result.Data,
				"summary": result.Summary,
			},
		})
	})

	// Desktop action — trigger write tool via risk preview (Task 14.2 + 14.3)
	r.Post("/api/v1/desktop/action/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")

		var args map[string]any
		if r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
				args = map[string]any{}
			}
		}
		if args == nil {
			args = map[string]any{}
		}

		// Verify tool exists and is a write tool
		tool, ok := registry.Get(name)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"code": 404, "error": fmt.Sprintf("action %q not found", name)})
			return
		}
		_ = tool // Could check tool.Type() == tools.ToolWrite

		// Create a preview (must be confirmed before execution)
		argsJSON, _ := json.Marshal(args)
		description := fmt.Sprintf("桌面操作: %s %s", name, string(argsJSON))
		p := previewEngine.Create(name+" "+string(argsJSON), description, "medium")

		// Task 14.3: write to session as system message
		sessions.GetOrCreate("desktop")
		sessions.AppendMessage("desktop", store.Message{
			Role:    "system",
			Content: fmt.Sprintf("[desktop_action] %s args=%s preview_id=%s triggered_by=ui_click", name, string(argsJSON), p.ID),
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"preview_id":  p.ID,
				"action":      name,
				"args":        args,
				"risk":        p.Risk,
				"expires_at":  p.ExpiresAt.Format(time.RFC3339),
				"triggered_by": "ui_click",
			},
		})
	})

	r.Post("/api/v1/safety/preview", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Command     string `json:"command"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"code":400,"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		// Validate the command first
		vr := safety.ValidateCommand(req.Command)
		risk := "low"
		if vr.Status == safety.StatusBlocked {
			risk = "blocked"
		}

		p := previewEngine.Create(req.Command, req.Description, risk)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"preview_id":  p.ID,
				"command":     p.Command,
				"description": p.Description,
				"risk":        p.Risk,
				"expires_at":  p.ExpiresAt.Format(time.RFC3339),
			},
		})
	})

	r.Post("/api/v1/safety/confirm", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PreviewID string `json:"preview_id"`
			Confirmed bool   `json:"confirmed"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"code":400,"error":"invalid body"}`, http.StatusBadRequest)
			return
		}

		p, err := previewEngine.Confirm(req.PreviewID, req.Confirmed)
		if err != nil {
			status := http.StatusBadRequest
			errCode := "API_INTERNAL_001"
			if err.Error() == "PREVIEW_EXPIRED_001" {
				status = http.StatusGone
				errCode = "PREVIEW_EXPIRED_001"
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]any{
				"code": status, "error": err.Error(), "error_code": errCode,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"preview_id": p.ID,
				"status":     string(p.Status),
				"command":    p.Command,
			},
		})
	})

	activeModelName := cfg.LLMModel
	if active, ok := modelPool.GetActive(); ok {
		activeModelName = active.Name + " (" + active.ModelID + ")"
	}
	fmt.Printf("ops-agent listening on :%s\n", cfg.Port)
	fmt.Printf("   LLM: %s @ %s\n", activeModelName, cfg.LLMBaseURL)
	fmt.Printf("   Tools: %d registered\n", len(registry.List()))
	fmt.Printf("   Model Pool: %d providers\n", len(modelPool.GetAll()))
	fmt.Printf("\n   Try: curl -X POST http://localhost:%s/api/v1/chat -H 'Content-Type: application/json' -d '{\"message\":\"看下磁盘\"}'\n", cfg.Port)

	// Serve frontend static files (production mode: serve from ./web/ directory)
	webDir := "./web"
	if _, err := os.Stat(webDir + "/index.html"); err == nil {
		// Serve static assets
		fileServer := http.FileServer(http.Dir(webDir))
		r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
			// Try to serve static file first
			path := req.URL.Path
			filePath := webDir + path
			if _, err := os.Stat(filePath); err == nil && path != "/" {
				fileServer.ServeHTTP(w, req)
				return
			}
			// Fallback to index.html (SPA routing)
			http.ServeFile(w, req, webDir+"/index.html")
		})
		fmt.Printf("   Web UI: http://localhost:%s (serving from %s)\n", cfg.Port, webDir)
	}

	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// sessionsToAgentMessages converts store messages to agent messages for context injection.
func sessionsToAgentMessages(msgs []store.Message) []agent.Message {
	result := make([]agent.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, agent.Message{Role: m.Role, Content: m.Content})
	}
	return result
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// auditBridge adapts audit.Writer to agent.AuditWriter interface.
type auditBridge struct {
	w *audit.Writer
}

func (b *auditBridge) Write(entry agent.AuditEntry) error {
	return b.w.Write(audit.Entry{
		TraceID:     entry.TraceID,
		SessionID:   entry.SessionID,
		RoundNumber: entry.RoundNumber,
		Stage:       audit.Stage(entry.Stage),
		Role:        entry.Role,
		Content:     entry.Content,
		TriggeredBy: entry.TriggeredBy,
		Status:      entry.Status,
		DurationMs:  entry.DurationMs,
	})
}
