package store

import (
	"sync"
	"time"
)

// Message represents a chat message in a session.
type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCalls  string `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// Session represents a conversation session.
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Messages  []Message `json:"messages"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

// SessionStore manages in-memory sessions (will migrate to SQLite in Task 10.2).
// Thread-safe for concurrent access from SSE handlers.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// GetOrCreate retrieves a session or creates a new one.
func (s *SessionStore) GetOrCreate(sessionID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()

	if sess, ok := s.sessions[sessionID]; ok {
		return sess
	}

	now := time.Now().Format(time.RFC3339)
	sess := &Session{
		ID:        sessionID,
		Title:     "新对话",
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.sessions[sessionID] = sess
	return sess
}

// AppendMessage adds a message to a session.
func (s *SessionStore) AppendMessage(sessionID string, msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return
	}
	if msg.CreatedAt == "" {
		msg.CreatedAt = time.Now().Format(time.RFC3339)
	}
	sess.Messages = append(sess.Messages, msg)
	sess.UpdatedAt = time.Now().Format(time.RFC3339)
}

// GetRecentMessages returns the last N messages for a session.
// Used to build context for multi-turn conversations.
func (s *SessionStore) GetRecentMessages(sessionID string, maxMessages int) []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil
	}

	msgs := sess.Messages
	if len(msgs) <= maxMessages {
		return msgs
	}
	return msgs[len(msgs)-maxMessages:]
}

// ListSessions returns all sessions sorted by updated_at desc.
func (s *SessionStore) ListSessions() []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		// Return without messages for list view
		result = append(result, Session{
			ID:        sess.ID,
			Title:     sess.Title,
			CreatedAt: sess.CreatedAt,
			UpdatedAt: sess.UpdatedAt,
		})
	}
	return result
}
