package audit

import (
	"database/sql"
	"encoding/json"
	"sync"
	"time"
)

// Stage represents the five-stage audit pipeline.
type Stage string

const (
	StageSense   Stage = "SENSE"
	StageAnalyze Stage = "ANALYZE"
	StagePlan    Stage = "PLAN"
	StageExecute Stage = "EXECUTE"
	StageOutput  Stage = "OUTPUT"
)

// Entry is a single audit log record.
type Entry struct {
	TraceID     string
	SessionID   string
	RoundNumber int
	Stage       Stage
	Role        string // multi-agent role (planner/executor/verifier)
	Content     any
	TriggeredBy string // llm | ui_click | terminal
	Status      string // ok | warning | blocked | error
	DurationMs  int
}

// Writer handles append-only audit log writes.
type Writer struct {
	db *sql.DB
	mu sync.Mutex
}

// NewWriter creates an audit writer (nil db = in-memory noop for dev).
func NewWriter(db *sql.DB) *Writer {
	return &Writer{db: db}
}

// Write appends an audit entry. Never updates or deletes.
func (w *Writer) Write(entry Entry) error {
	if w.db == nil {
		return nil // dev mode: no persistence
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	contentJSON, _ := json.Marshal(entry.Content)
	if entry.TriggeredBy == "" {
		entry.TriggeredBy = "llm"
	}
	if entry.Status == "" {
		entry.Status = "ok"
	}
	if entry.RoundNumber == 0 {
		entry.RoundNumber = 1
	}

	_, err := w.db.Exec(`INSERT INTO audit_logs 
		(trace_id, session_id, round_number, stage, role, content, triggered_by, status, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.TraceID, entry.SessionID, entry.RoundNumber,
		string(entry.Stage), entry.Role, string(contentJSON),
		entry.TriggeredBy, entry.Status, entry.DurationMs,
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	)
	return err
}

// Query returns recent audit entries for a session.
func (w *Writer) Query(sessionID string, limit int) ([]map[string]any, error) {
	if w.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	rows, err := w.db.Query(`SELECT trace_id, stage, role, content, status, duration_ms, created_at 
		FROM audit_logs WHERE session_id = ? ORDER BY created_at DESC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		var traceID, stage, role, content, status, createdAt string
		var durationMs int
		if err := rows.Scan(&traceID, &stage, &role, &content, &status, &durationMs, &createdAt); err != nil {
			continue
		}
		results = append(results, map[string]any{
			"trace_id":    traceID,
			"stage":       stage,
			"role":        role,
			"content":     content,
			"status":      status,
			"duration_ms": durationMs,
			"created_at":  createdAt,
		})
	}
	return results, nil
}
