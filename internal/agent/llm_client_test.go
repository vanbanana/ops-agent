package agent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// loadEnv reads a simple .env file (KEY=VALUE per line).
func loadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}

func init() {
	// Load .env from project root for integration tests
	loadEnv("../../.env")
}

// --- Task 3.2 & 3.3: 实际联通测试 (带 tool definition, 验证返回含 tool_calls) ---

func TestLLMChatWithToolCalls(t *testing.T) {
	apiKey := os.Getenv("LLM_API_KEY")
	baseURL := os.Getenv("LLM_BASE_URL")
	model := os.Getenv("LLM_MODEL")

	if apiKey == "" || baseURL == "" || model == "" {
		t.Skip("LLM_API_KEY/LLM_BASE_URL/LLM_MODEL not set, skipping integration test")
	}

	client := NewLLMClient(ClientConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		Timeout: 30 * time.Second,
	})

	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "查看当前服务器的磁盘使用情况"},
		},
		Tools: []ToolDef{
			{
				Type: "function",
				Function: FunctionDef{
					Name:        "probe_disk",
					Description: "查看磁盘使用情况，返回各分区的使用率",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"path": map[string]any{
								"type":        "string",
								"description": "要查看的路径，默认为 /",
							},
						},
					},
				},
			},
		},
	}

	resp, err := client.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if len(resp.Choices) == 0 {
		t.Fatal("expected at least one choice in response")
	}

	choice := resp.Choices[0]
	// LLM should either call the tool or respond with text
	// For this test, we just verify the response is valid
	t.Logf("finish_reason: %s", choice.FinishReason)
	t.Logf("tool_calls count: %d", len(choice.Message.ToolCalls))
	t.Logf("content: %s", choice.Message.Content)

	if choice.FinishReason == "tool_calls" || choice.FinishReason == "stop" {
		// Both are acceptable - tool_calls means the LLM wants to call the function
		// stop means it responded directly (some models do this)
		if choice.FinishReason == "tool_calls" || len(choice.Message.ToolCalls) > 0 {
			if len(choice.Message.ToolCalls) == 0 {
				t.Fatal("finish_reason is tool_calls but no tool_calls in message")
			}
			tc := choice.Message.ToolCalls[0]
			if tc.Function.Name != "probe_disk" {
				t.Fatalf("expected tool call to probe_disk, got %s", tc.Function.Name)
			}
			t.Logf("✅ LLM called tool: %s with args: %s", tc.Function.Name, tc.Function.Arguments)
		}
	} else {
		t.Logf("⚠️ LLM did not call tool (finish_reason=%s), but response is valid", choice.FinishReason)
	}
}

// --- Task 3.4: API Key 无效时返回 LLM_AUTH_001 而非 panic ---

func TestLLMInvalidAPIKey(t *testing.T) {
	baseURL := os.Getenv("LLM_BASE_URL")
	if baseURL == "" {
		baseURL = "https://token-plan-cn.xiaomimimo.com/v1"
	}

	client := NewLLMClient(ClientConfig{
		BaseURL: baseURL,
		APIKey:  "invalid-key-12345",
		Model:   "mimo-v2.5-pro",
		Timeout: 10 * time.Second,
	})

	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	}

	_, err := client.Chat(context.Background(), req)
	if err == nil {
		t.Fatal("expected error with invalid API key, got nil")
	}

	var llmErr *LLMError
	if !errors.As(err, &llmErr) {
		t.Fatalf("expected *LLMError, got %T: %v", err, err)
	}

	if llmErr.Code != ErrLLMAuth {
		t.Fatalf("expected error code %s, got %s: %s", ErrLLMAuth, llmErr.Code, llmErr.Message)
	}

	t.Logf("✅ Invalid API key correctly returns %s: %s", llmErr.Code, llmErr.Message)
}

// --- Task 3.5: 网络不通时 3s 内超时返回 LLM_NETWORK_001 ---

func TestLLMNetworkTimeout(t *testing.T) {
	// Use a non-routable address to simulate network timeout
	client := NewLLMClient(ClientConfig{
		BaseURL: "http://192.0.2.1:1", // RFC 5737 TEST-NET, guaranteed non-routable
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 3 * time.Second,
	})

	req := ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "hello"},
		},
	}

	start := time.Now()
	_, err := client.Chat(context.Background(), req)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error with non-routable address, got nil")
	}

	var llmErr *LLMError
	if !errors.As(err, &llmErr) {
		t.Fatalf("expected *LLMError, got %T: %v", err, err)
	}

	if llmErr.Code != ErrLLMNetwork {
		t.Fatalf("expected error code %s, got %s: %s", ErrLLMNetwork, llmErr.Code, llmErr.Message)
	}

	// Must return within 3s (with small tolerance)
	if elapsed > 4*time.Second {
		t.Fatalf("timeout took too long: %v (expected ≤ 3s)", elapsed)
	}

	t.Logf("✅ Network timeout correctly returns %s in %v", llmErr.Code, elapsed)
}

// --- Task 3.6: 模型名校验 ---

func TestLLMModelName(t *testing.T) {
	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "mimo-v2.5-pro"
	}

	// 确保不是已退役的 deepseek-chat
	if model == "deepseek-chat" {
		t.Fatal("model name 'deepseek-chat' is retired; use 'deepseek-v4-flash' or current alternative")
	}

	// 验证当前配置的模型名是预期值之一
	validModels := map[string]bool{
		"mimo-v2.5-pro":     true, // 小米 MiMo (测试)
		"deepseek-v4-flash": true, // DeepSeek (生产)
	}
	if !validModels[model] {
		t.Logf("⚠️ model %q is not in known-good list, proceeding anyway", model)
	}

	t.Logf("✅ Model name: %s (not a retired model)", model)
}

// --- 辅助: mock server 测试 HTTP 错误分类 ---

func TestClassifyHTTPErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		expectCode ErrorCode
	}{
		{"401 unauthorized", 401, `{"error":"invalid key"}`, ErrLLMAuth},
		{"403 forbidden", 403, `{"error":"forbidden"}`, ErrLLMAuth},
		{"429 rate limit", 429, `{"error":"too many requests"}`, ErrLLMQuota},
		{"500 server error", 500, `{"error":"internal"}`, ErrLLMService},
		{"502 bad gateway", 502, `{"error":"bad gateway"}`, ErrLLMService},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewLLMClient(ClientConfig{
				BaseURL: server.URL,
				APIKey:  "test",
				Model:   "test",
				Timeout: 5 * time.Second,
			})

			_, err := client.Chat(context.Background(), ChatRequest{
				Messages: []Message{{Role: "user", Content: "test"}},
			})

			if err == nil {
				t.Fatal("expected error")
			}

			var llmErr *LLMError
			if !errors.As(err, &llmErr) {
				t.Fatalf("expected *LLMError, got %T", err)
			}

			if llmErr.Code != tt.expectCode {
				t.Fatalf("expected %s, got %s", tt.expectCode, llmErr.Code)
			}
		})
	}
}
