package safety

import (
	"fmt"
	"sync"
	"time"
)

// PreviewStatus tracks the state of a risk preview.
type PreviewStatus string

const (
	PreviewPending   PreviewStatus = "pending"
	PreviewConfirmed PreviewStatus = "confirmed"
	PreviewCancelled PreviewStatus = "cancelled"
	PreviewExpired   PreviewStatus = "expired"
)

// Preview represents a pending dangerous operation awaiting confirmation.
type Preview struct {
	ID          string        `json:"preview_id"`
	Command     string        `json:"command"`
	Description string        `json:"description"`
	Risk        string        `json:"risk"`
	Status      PreviewStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
}

const PreviewTTL = 5 * time.Minute

// PreviewEngine manages risk previews with expiration.
type PreviewEngine struct {
	mu       sync.RWMutex
	previews map[string]*Preview
}

// NewPreviewEngine creates a new preview engine.
func NewPreviewEngine() *PreviewEngine {
	return &PreviewEngine{
		previews: make(map[string]*Preview),
	}
}

// Create generates a new preview for a risky command.
func (pe *PreviewEngine) Create(command, description, risk string) *Preview {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	now := time.Now()
	p := &Preview{
		ID:          fmt.Sprintf("prv_%d", now.UnixNano()),
		Command:     command,
		Description: description,
		Risk:        risk,
		Status:      PreviewPending,
		CreatedAt:   now,
		ExpiresAt:   now.Add(PreviewTTL),
	}
	pe.previews[p.ID] = p
	return p
}

// Confirm confirms or cancels a preview. Returns the preview and error.
func (pe *PreviewEngine) Confirm(previewID string, confirmed bool) (*Preview, error) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	p, ok := pe.previews[previewID]
	if !ok {
		return nil, fmt.Errorf("preview not found: %s", previewID)
	}

	// Check expiration
	if time.Now().After(p.ExpiresAt) {
		p.Status = PreviewExpired
		return p, fmt.Errorf("PREVIEW_EXPIRED_001")
	}

	if p.Status != PreviewPending {
		return p, fmt.Errorf("preview already %s", p.Status)
	}

	if confirmed {
		p.Status = PreviewConfirmed
	} else {
		p.Status = PreviewCancelled
	}

	return p, nil
}

// Get retrieves a preview by ID.
func (pe *PreviewEngine) Get(previewID string) (*Preview, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	p, ok := pe.previews[previewID]
	return p, ok
}
