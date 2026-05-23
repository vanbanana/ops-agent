package store

import (
	"database/sql"
	"fmt"
	"time"
)

// SQLiteSessionStore persists sessions to SQLite.
type SQLiteSessionStore struct {
	db *sql.DB
}

// NewSQLiteSessionStore creates a persistent session store.
func NewSQLiteSessionStore(db *sql.DB) *SQLiteSessionStore {
	return &SQLiteSessionStore{db: db}
}

// GetOrCreate retrieves or creates a session in SQLite.
func (s *SQLiteSessionStore) GetOrCreate(sessionID string) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	s.db.Exec(`INSERT OR IGNORE INTO sessions (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		sessionID, "新对话", now, now)
}

// AppendMessage persists a message to SQLite.
func (s *SQLiteSessionStore) AppendMessage(sessionID string, msg Message) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	s.db.Exec(`INSERT INTO messages (id, session_id, role, content, created_at) VALUES (?, ?, ?, ?, ?)`,
		msgID, sessionID, msg.Role, msg.Content, now)
	s.db.Exec(`UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
}

// GetRecentMessages loads recent messages from SQLite.
func (s *SQLiteSessionStore) GetRecentMessages(sessionID string, maxMessages int) []Message {
	rows, err := s.db.Query(`SELECT role, content, created_at FROM messages 
		WHERE session_id = ? ORDER BY created_at DESC LIMIT ?`, sessionID, maxMessages)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.Role, &m.Content, &m.CreatedAt); err != nil {
			continue
		}
		msgs = append(msgs, m)
	}
	// Reverse (DB returns DESC, we need ASC)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs
}

// ListSessions returns all sessions.
func (s *SQLiteSessionStore) ListSessions() []Session {
	rows, err := s.db.Query(`SELECT id, title, created_at, updated_at FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sess Session
		if err := rows.Scan(&sess.ID, &sess.Title, &sess.CreatedAt, &sess.UpdatedAt); err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions
}
