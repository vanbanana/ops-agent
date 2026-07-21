package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"ops-agent/internal/agent"
	"ops-agent/internal/config"
	"ops-agent/internal/llm"
	"ops-agent/internal/permission"
	"ops-agent/internal/store"
	"ops-agent/internal/tools"
)

// App holds the shared runtime for the OpenCode-aligned CLI.
type App struct {
	cfg       *config.Config
	db        *sql.DB
	pool      *agent.ModelPool
	registry  tools.ToolRegistry
	sessions  store.SessionStoreInterface
	permSvc   *permission.Service
	llmClient *agent.HotReloadClient
}

// newApp loads config, opens the database, initializes the model pool,
// tool registry, session store, and permission service.
func newApp() (*App, error) {
	config.LoadDotEnv(".env")

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	var db *sql.DB
	if db, err = store.OpenDB(cfg.DBPath); err != nil {
		log.Printf("warning: SQLite unavailable (%v), sessions will be in-memory", err)
	}

	pool := agent.NewModelPool(db, "providers.json", cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel)

	clientCfg := agent.ClientConfig{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
		Timeout: 60 * time.Second,
	}
	llmClient := agent.NewHotReloadClient(clientCfg)

	// If the model pool already selected a different active provider, use it.
	if active, ok := pool.GetActive(); ok && active.ModelID != cfg.LLMModel {
		llmClient.Reload(agent.ClientConfig{
			BaseURL: active.BaseURL,
			APIKey:  active.APIKey,
			Model:   active.ModelID,
			Timeout: 60 * time.Second,
		})
	}

	pool.SetOnChange(func(active llm.Provider) {
		llmClient.Reload(agent.ClientConfig{
			BaseURL: active.BaseURL,
			APIKey:  active.APIKey,
			Model:   active.ModelID,
			Timeout: 60 * time.Second,
		})
		log.Printf("model switched to: %s (%s)", active.Name, active.ModelID)
	})

	registry := tools.NewRegistry()
	tools.RegisterAllProbes(registry)
	tools.RegisterWriteTools(registry)

	var sessions store.SessionStoreInterface
	if db != nil {
		sessions = store.NewSQLiteSessionStore(db)
	} else {
		sessions = store.NewSessionStore()
	}

	permSvc := permission.NewService()

	return &App{
		cfg:       cfg,
		db:        db,
		pool:      pool,
		registry:  registry,
		sessions:  sessions,
		permSvc:   permSvc,
		llmClient: llmClient,
	}, nil
}

// close releases database resources.
func (a *App) close() {
	if a.db != nil {
		a.db.Close()
	}
}

// activeModel returns the currently active model ID.
func (a *App) activeModel() string {
	if active, ok := a.pool.GetActive(); ok {
		return active.ModelID
	}
	return a.cfg.LLMModel
}

// switchModel tries to switch to the provider matching modelID (id or model_id).
func (a *App) switchModel(modelID string) error {
	for _, p := range a.pool.GetAllFull() {
		if p.ID == modelID || p.ModelID == modelID {
			a.llmClient.Reload(agent.ClientConfig{
				BaseURL: p.BaseURL,
				APIKey:  p.APIKey,
				Model:   p.ModelID,
				Timeout: 60 * time.Second,
			})
			return nil
		}
	}
	return fmt.Errorf("model %q not found in provider pool", modelID)
}

// newAgent creates a fresh agent wired with the current runtime.
func (a *App) newAgent() *agent.Agent {
	ag := agent.NewAgent(a.llmClient, a.registry, agent.AgentConfig{
		Model: a.activeModel(),
	}, a.permSvc)
	tools.RegisterMultiAgentTool(a.registry, ag.MultiAgent())
	return ag
}

// resolveSession picks a session id based on --continue / --session flags.
func (a *App) resolveSession(cont, sessionID string) (string, error) {
	switch {
	case sessionID != "":
		return sessionID, nil
	case cont != "":
		// "last" or any non-empty value means resume the most recent session.
		sessions := a.sessions.ListSessions()
		if len(sessions) == 0 {
			return "", fmt.Errorf("no sessions to continue")
		}
		return sessions[0].ID, nil
	default:
		return fmt.Sprintf("sess_%d", time.Now().UnixNano()), nil
	}
}

// runChat executes a single chat turn, streaming events to the console.
func (a *App) runChat(ctx context.Context, sessionID, message string, render *renderer) error {
	a.sessions.GetOrCreate(sessionID)
	history := storeToAgentMessages(a.sessions.GetRecentMessages(sessionID, a.cfg.MaxHistoryMessages))
	a.sessions.AppendMessage(sessionID, store.Message{Role: "user", Content: message})

	ag := a.newAgent()
	events := ag.RunStream(ctx, sessionID, message, history)

	reply := ""
	for event := range events {
		if event.Type == "permission_request" {
			if err := a.handlePermission(event.Data); err != nil {
				return err
			}
			continue
		}
		render.Event(event)
		if event.Type == "output" {
			if r, ok := event.Data["reply"].(string); ok {
				reply = r
			}
		}
	}

	if reply != "" {
		a.sessions.AppendMessage(sessionID, store.Message{Role: "assistant", Content: reply})
	}
	return nil
}

// handlePermission asks the terminal user to allow/deny a pending permission request.
func (a *App) handlePermission(data map[string]any) error {
	reqID, _ := data["request_id"].(string)
	toolName, _ := data["tool"].(string)
	command, _ := data["command"].(string)
	risk, _ := data["risk_level"].(string)

	fmt.Printf("\n%s⚠️  Permission request%s\n", colorYellow, colorReset)
	fmt.Printf("   Tool: %s\n", toolName)
	fmt.Printf("   Command: %s\n", command)
	fmt.Printf("   Risk: %s\n", risk)
	fmt.Printf("   Allow? [y=once / a=always this session / n=deny]: ")

	reader := bufioReader
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read permission response: %w", err)
	}
	line = strings.ToLower(strings.TrimSpace(line))

	action := "deny"
	switch line {
	case "y", "yes", "once":
		action = "allow"
	case "a", "always", "allow_session":
		action = "allow_session"
	case "n", "no", "deny":
		action = "deny"
	default:
		fmt.Println("   Unrecognized response, defaulting to deny.")
		action = "deny"
	}

	if err := a.permSvc.Respond(reqID, action); err != nil {
		return fmt.Errorf("permission respond: %w", err)
	}
	return nil
}
