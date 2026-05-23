package agent

import (
	"context"
	"testing"
	"time"
)

// --- Mock: Verifier returns garbage (non-JSON) ---

type mockLLMVerifierGarbage struct {
	callCount int
}

func (m *mockLLMVerifierGarbage) Chat(_ context.Context, req ChatRequest) (*ChatResponse, error) {
	m.callCount++
	// Planner call: return valid subtasks
	if m.callCount == 1 {
		return &ChatResponse{
			Choices: []Choice{{
				Message:      Message{Role: "assistant", Content: `[{"id":"1","description":"检查磁盘","tools":"probe_disk"}]`},
				FinishReason: "stop",
			}},
			Usage: Usage{PromptTokens: 100, CompletionTokens: 50},
		}, nil
	}
	// Executor call: return text (no tool calls)
	if m.callCount == 2 {
		return &ChatResponse{
			Choices: []Choice{{
				Message:      Message{Role: "assistant", Content: "磁盘使用 87%"},
				FinishReason: "stop",
			}},
		}, nil
	}
	// Verifier call: return GARBAGE (non-JSON)
	if m.callCount == 3 {
		return &ChatResponse{
			Choices: []Choice{{
				Message:      Message{Role: "assistant", Content: "这不是JSON，我是乱码 }{]["},
				FinishReason: "stop",
			}},
		}, nil
	}
	// Synthesize call
	return &ChatResponse{
		Choices: []Choice{{
			Message:      Message{Role: "assistant", Content: "综合结论: 磁盘使用率高"},
			FinishReason: "stop",
		}},
	}, nil
}

// --- Mock: Verifier always returns false ---

type mockLLMVerifierAlwaysFalse struct {
	callCount int
}

func (m *mockLLMVerifierAlwaysFalse) Chat(_ context.Context, req ChatRequest) (*ChatResponse, error) {
	m.callCount++

	// Alternate between planner, executor, verifier across 3 iterations
	// Simple pattern: planner returns subtask, executor returns finding, verifier says false
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if containsSubstr(msg.Content, "Planner") || containsSubstr(msg.Content, "拆解") {
				return &ChatResponse{
					Choices: []Choice{{
						Message:      Message{Role: "assistant", Content: `[{"id":"1","description":"检查","tools":"probe_disk"}]`},
						FinishReason: "stop",
					}},
					Usage: Usage{PromptTokens: 50, CompletionTokens: 30},
				}, nil
			}
			if containsSubstr(msg.Content, "Verifier") || containsSubstr(msg.Content, "验证") {
				return &ChatResponse{
					Choices: []Choice{{
						Message:      Message{Role: "assistant", Content: `{"verified":false,"reason":"信息不足","confidence":0.3,"missing_info":["需要更多数据"]}`},
						FinishReason: "stop",
					}},
				}, nil
			}
			if containsSubstr(msg.Content, "综合") || containsSubstr(msg.Content, "整合") {
				return &ChatResponse{
					Choices: []Choice{{
						Message:      Message{Role: "assistant", Content: "尽力分析结果"},
						FinishReason: "stop",
					}},
				}, nil
			}
		}
	}

	// Default: executor returns text
	return &ChatResponse{
		Choices: []Choice{{
			Message:      Message{Role: "assistant", Content: "执行结果: 数据正常"},
			FinishReason: "stop",
		}},
	}, nil
}

// --- Task 11.9: Verifier 返回乱码 → 降级 verified=false 不 panic ---

func TestMultiAgentVerifierGarbage(t *testing.T) {
	llm := &mockLLMVerifierGarbage{}
	reg := &mockRegistry{}
	ma := NewMultiAgent(llm, reg, AgentConfig{Model: "test"})

	out := make(chan Event, 128)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run in goroutine, it will close 'out' when done via the caller pattern
	go func() {
		defer close(out)
		ma.Run(ctx, "sess-test", "trc-test", "看磁盘", out)
	}()

	var hasVerifierResult bool
	var verified bool
	var hasOutput bool
	for ev := range out {
		if ev.Type == "verifier_result" {
			hasVerifierResult = true
			if v, ok := ev.Data["verified"].(bool); ok {
				verified = v
			}
		}
		if ev.Type == "output" {
			hasOutput = true
		}
	}

	if !hasVerifierResult {
		t.Fatal("expected verifier_result event")
	}
	// Garbage JSON → verified should be false (graceful degradation)
	if verified {
		t.Fatal("garbage verifier response should result in verified=false")
	}
	if !hasOutput {
		t.Fatal("should still produce output even with garbage verifier")
	}

	t.Log("✅ Verifier garbage JSON degrades gracefully to verified=false, no panic")
}

// --- Task 11.10: Verifier 永远 false → 3 轮后给 best-effort ---

func TestMultiAgentVerifierAlwaysFalse(t *testing.T) {
	llm := &mockLLMVerifierAlwaysFalse{}
	reg := &mockRegistry{}
	ma := NewMultiAgent(llm, reg, AgentConfig{Model: "test"})

	out := make(chan Event, 256)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	go func() {
		defer close(out)
		ma.Run(ctx, "sess-test", "trc-test", "全面分析服务器", out)
	}()

	verifierCount := 0
	var finalReply string
	var hasDone bool
	for ev := range out {
		if ev.Type == "verifier_result" {
			verifierCount++
		}
		if ev.Type == "output" {
			if r, ok := ev.Data["reply"].(string); ok {
				finalReply = r
			}
		}
		if ev.Type == "done" {
			hasDone = true
		}
	}

	if verifierCount < MaxMultiIterations {
		t.Fatalf("expected at least %d verifier results (max iterations), got %d", MaxMultiIterations, verifierCount)
	}
	if finalReply == "" {
		t.Fatal("should produce best-effort reply after max iterations")
	}
	if !hasDone {
		t.Fatal("should emit done event")
	}

	t.Logf("✅ Verifier always-false → %d iterations → best-effort reply produced", verifierCount)
}

// --- Task 11.13: Coordinator 显示等待进度 ---

func TestMultiAgentCoordinatorProgress(t *testing.T) {
	llm := &mockLLMVerifierGarbage{} // reuse, it goes through planner→executor→verifier
	reg := &mockRegistry{}
	ma := NewMultiAgent(llm, reg, AgentConfig{Model: "test"})

	out := make(chan Event, 128)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		defer close(out)
		ma.Run(ctx, "sess-test", "trc-test", "分析服务器", out)
	}()

	var coordinatorEvents []map[string]any
	for ev := range out {
		if ev.Type == "agent_role" {
			if role, ok := ev.Data["role"].(string); ok && role == "coordinator" {
				coordinatorEvents = append(coordinatorEvents, ev.Data)
			}
		}
	}

	if len(coordinatorEvents) == 0 {
		t.Fatal("expected coordinator events showing progress")
	}

	// Check at least one has "action" field
	hasAction := false
	for _, ce := range coordinatorEvents {
		if _, ok := ce["action"]; ok {
			hasAction = true
			break
		}
	}
	if !hasAction {
		t.Fatal("coordinator events should have action field (dispatch/waiting/collected)")
	}

	t.Logf("✅ Coordinator emits %d progress events with action field", len(coordinatorEvents))
}

// Reuse helper — uses indexOf for string containment
func containsSubstr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexOfStr(s, sub) >= 0)
}

func indexOfStr(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// ChatStream implementations for multi_test mocks
func (m *mockLLMVerifierGarbage) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	return mockStreamFromChat(m, ctx, req)
}

func (m *mockLLMVerifierAlwaysFalse) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	return mockStreamFromChat(m, ctx, req)
}
