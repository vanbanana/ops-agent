package store

import (
	"fmt"
	"sync"
	"testing"
)

// --- Task 10.15: 同一 session 连续两句对话, 第二句能引用第一句的上下文 ---

func TestMultiTurnContext(t *testing.T) {
	store := NewSessionStore()

	sessionID := "sess_multi_turn_test"
	store.GetOrCreate(sessionID)

	// First turn: user asks about disk
	store.AppendMessage(sessionID, Message{Role: "user", Content: "磁盘还剩多少"})
	store.AppendMessage(sessionID, Message{Role: "assistant", Content: "根分区使用 87%，剩余 520MB"})

	// Second turn: user asks follow-up
	store.AppendMessage(sessionID, Message{Role: "user", Content: "那 /var 呢"})

	// Get context — should contain all 3 messages
	history := store.GetRecentMessages(sessionID, 10)
	if len(history) != 3 {
		t.Fatalf("expected 3 messages in history, got %d", len(history))
	}

	// Verify order and content
	if history[0].Role != "user" || history[0].Content != "磁盘还剩多少" {
		t.Fatalf("first message wrong: %+v", history[0])
	}
	if history[1].Role != "assistant" || history[1].Content != "根分区使用 87%，剩余 520MB" {
		t.Fatalf("second message wrong: %+v", history[1])
	}
	if history[2].Role != "user" || history[2].Content != "那 /var 呢" {
		t.Fatalf("third message wrong: %+v", history[2])
	}

	t.Logf("✅ Multi-turn context preserved: 3 messages in correct order")
}

// --- Test: GetRecentMessages with limit ---

func TestGetRecentMessagesLimit(t *testing.T) {
	store := NewSessionStore()
	sessionID := "sess_limit_test"
	store.GetOrCreate(sessionID)

	// Add 20 messages
	for i := 0; i < 20; i++ {
		store.AppendMessage(sessionID, Message{Role: "user", Content: fmt.Sprintf("msg_%d", i)})
	}

	// Get last 5
	history := store.GetRecentMessages(sessionID, 5)
	if len(history) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(history))
	}

	// Should be the LAST 5 (msg_15 through msg_19)
	if history[0].Content != "msg_15" {
		t.Fatalf("expected first of recent 5 to be msg_15, got %q", history[0].Content)
	}
	if history[4].Content != "msg_19" {
		t.Fatalf("expected last of recent 5 to be msg_19, got %q", history[4].Content)
	}

	t.Logf("✅ GetRecentMessages correctly returns last N messages")
}

// --- Test: Empty session returns nil ---

func TestGetRecentMessagesNonExistent(t *testing.T) {
	store := NewSessionStore()
	history := store.GetRecentMessages("nonexistent", 10)
	if history != nil {
		t.Fatalf("expected nil for non-existent session, got %v", history)
	}
}

// --- Test: Concurrent access safety ---

func TestSessionStoreConcurrency(t *testing.T) {
	store := NewSessionStore()
	sessionID := "sess_concurrent"
	store.GetOrCreate(sessionID)

	var wg sync.WaitGroup
	// 100 goroutines writing simultaneously
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			store.AppendMessage(sessionID, Message{Role: "user", Content: fmt.Sprintf("concurrent_%d", n)})
		}(i)
	}
	wg.Wait()

	history := store.GetRecentMessages(sessionID, 200)
	if len(history) != 100 {
		t.Fatalf("expected 100 messages after concurrent writes, got %d", len(history))
	}

	t.Logf("✅ SessionStore handles 100 concurrent writes safely")
}

// --- Test: GetOrCreate is idempotent ---

func TestGetOrCreateIdempotent(t *testing.T) {
	store := NewSessionStore()
	s1 := store.GetOrCreate("sess_idem")
	store.AppendMessage("sess_idem", Message{Role: "user", Content: "hello"})
	s2 := store.GetOrCreate("sess_idem")

	if s1.ID != s2.ID {
		t.Fatal("GetOrCreate should return same session")
	}
	if len(s2.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(s2.Messages))
	}
}
