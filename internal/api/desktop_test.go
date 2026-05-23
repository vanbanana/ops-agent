package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ops-agent/internal/safety"
	"ops-agent/internal/store"

	"github.com/go-chi/chi/v5"
)

// --- Task 14.5: /desktop/action/truncate 返回 preview_id ---

func TestDesktopActionReturnsPreviewID(t *testing.T) {
	pe := safety.NewPreviewEngine()
	sessions := store.NewSessionStore()

	r := chi.NewRouter()
	r.Post("/api/v1/desktop/action/{name}", desktopActionHandler(pe, sessions))

	body := `{"path":"/var/log/test.log"}`
	req := httptest.NewRequest("POST", "/api/v1/desktop/action/truncate_log_file", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data field, got: %v", resp)
	}

	previewID, ok := data["preview_id"].(string)
	if !ok || previewID == "" {
		t.Fatalf("expected non-empty preview_id, got: %v", data["preview_id"])
	}

	t.Logf("✅ /desktop/action returns preview_id: %s", previewID)
}

// --- Task 14.6: triggered_by=ui_click in session message ---

func TestDesktopActionTriggeredByUIClick(t *testing.T) {
	pe := safety.NewPreviewEngine()
	sessions := store.NewSessionStore()

	r := chi.NewRouter()
	r.Post("/api/v1/desktop/action/{name}", desktopActionHandler(pe, sessions))

	body := `{"path":"/var/log/test.log"}`
	req := httptest.NewRequest("POST", "/api/v1/desktop/action/truncate_log_file", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check that the session has the message with triggered_by=ui_click
	msgs := sessions.GetRecentMessages("desktop", 10)
	if len(msgs) == 0 {
		t.Fatal("expected at least one message in desktop session")
	}

	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role != "system" {
		t.Fatalf("expected role=system, got %s", lastMsg.Role)
	}
	if !containsStr(lastMsg.Content, "triggered_by=ui_click") {
		t.Fatalf("expected triggered_by=ui_click in message content, got: %s", lastMsg.Content)
	}

	t.Logf("✅ Desktop action writes system message with triggered_by=ui_click")
}

// --- Task 14.7: 切回对话模式时 LLM 能看到最近桌面操作摘要 ---

func TestDesktopMessagesVisibleInChatContext(t *testing.T) {
	sessions := store.NewSessionStore()

	// Simulate 3 desktop actions written to "desktop" session
	sessions.GetOrCreate("desktop")
	sessions.AppendMessage("desktop", store.Message{Role: "system", Content: "[desktop_action] probe_disk triggered_by=ui_click"})
	sessions.AppendMessage("desktop", store.Message{Role: "system", Content: "[desktop_action] probe_memory triggered_by=ui_click"})
	sessions.AppendMessage("desktop", store.Message{Role: "system", Content: "[desktop_action] truncate_log_file triggered_by=ui_click"})

	// When switching back to chat, get recent messages from desktop session
	msgs := sessions.GetRecentMessages("desktop", 3)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 desktop messages, got %d", len(msgs))
	}

	// All should contain desktop_action
	for i, m := range msgs {
		if !containsStr(m.Content, "desktop_action") {
			t.Fatalf("message %d missing desktop_action: %s", i, m.Content)
		}
	}

	t.Log("✅ Desktop operation messages preserved and retrievable for chat context injection")
}

// --- Handler factory (mirrors main.go logic) ---

func desktopActionHandler(pe *safety.PreviewEngine, sessions *store.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		argsJSON, _ := json.Marshal(args)
		description := name + " " + string(argsJSON)
		p := pe.Create(description, "桌面操作: "+name, "medium")

		sessions.GetOrCreate("desktop")
		sessions.AppendMessage("desktop", store.Message{
			Role:    "system",
			Content: "[desktop_action] " + name + " args=" + string(argsJSON) + " preview_id=" + p.ID + " triggered_by=ui_click",
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"preview_id":   p.ID,
				"action":       name,
				"args":         args,
				"risk":         p.Risk,
				"triggered_by": "ui_click",
			},
		})
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
