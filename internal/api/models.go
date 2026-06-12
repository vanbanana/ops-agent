package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ops-agent/internal/llm"
)

// ModelsHandler manages the model pool API endpoints.
type ModelsHandler struct {
	Pool *llm.ModelPool
}

// HandleGetPool returns all configured providers (GET /api/v1/models/pool).
func (h *ModelsHandler) HandleGetPool(w http.ResponseWriter, r *http.Request) {
	providers := h.Pool.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"code": 0,
		"data": map[string]any{
			"providers": providers,
			"count":     len(providers),
		},
	})
}

// HandleSavePool saves the entire pool (PUT /api/v1/models/pool).
func (h *ModelsHandler) HandleSavePool(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Providers []llm.Provider `json:"providers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request body"})
		return
	}

	if len(req.Providers) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "at least one provider is required"})
		return
	}

	// Validate each provider
	for i, p := range req.Providers {
		if p.ID == "" || p.BaseURL == "" || p.ModelID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"code":  400,
				"error": fmt.Sprintf("provider[%d]: id, base_url, model_id are required", i),
			})
			return
		}
	}

	if err := h.Pool.SavePool(req.Providers); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"code": 500, "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": map[string]any{"status": "ok"}})
}

// HandleSwitch switches the active model (POST /api/v1/models/switch).
func (h *ModelsHandler) HandleSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request body"})
		return
	}
	if req.ID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "id is required"})
		return
	}

	if err := h.Pool.Switch(req.ID); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"code": 404, "error": err.Error()})
		return
	}

	active, _ := h.Pool.GetActivePublic()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"code": 0,
		"data": map[string]any{
			"active": active,
		},
	})
}

// HandleTest tests connectivity to a specific provider (POST /api/v1/models/test).
func (h *ModelsHandler) HandleTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string `json:"id"`
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		ModelID string `json:"model_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request body"})
		return
	}

	// If ID is provided, look up from pool; otherwise use provided fields directly
	baseURL := req.BaseURL
	apiKey := req.APIKey
	modelID := req.ModelID

	if req.ID != "" && (baseURL == "" || apiKey == "" || modelID == "") {
		prov, ok := h.Pool.GetProviderByID(req.ID)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"code": 404, "error": "provider not found"})
			return
		}
		if baseURL == "" {
			baseURL = prov.BaseURL
		}
		if apiKey == "" {
			apiKey = prov.APIKey
		}
		if modelID == "" {
			modelID = prov.ModelID
		}
	}

	if baseURL == "" || apiKey == "" || modelID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "base_url, api_key, model_id are required (or provide an existing provider id)"})
		return
	}

	// Send a minimal ping request
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	pingBody, _ := json.Marshal(map[string]any{
		"model":    modelID,
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
		"stream":   false,
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(pingBody))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"success": false, "error": err.Error(), "latency_ms": 0},
		})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"success": false, "error": err.Error(), "latency_ms": latency},
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 200))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{"success": false, "error": errMsg, "latency_ms": latency},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"code": 0,
		"data": map[string]any{"success": true, "latency_ms": latency},
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
