package permission

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Mode defines the permission enforcement level.
type Mode string

const (
	ModeDefault     Mode = "default"      // Write ops require confirmation
	ModeAutoApprove Mode = "auto_approve" // All ops auto-approved
)

// Request holds a permission request sent to the frontend.
type Request struct {
	ID          string `json:"request_id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool"`
	Command     string `json:"command"`
	RiskLevel   string `json:"risk_level"`
	Description string `json:"description"`
	ExpiresAt   string `json:"expires_at"`
}

// NotifyFunc is called to push a permission_request SSE event to the client.
// The loop passes its own out-channel wrapped in this closure.
type NotifyFunc func(req Request)

// cachedPermission stores a session-level grant.
type cachedPermission struct {
	SessionID string
	ToolName  string
	Command   string
}

// Service manages permission requests with channel-based blocking.
type Service struct {
	mu              sync.RWMutex
	mode            Mode
	pendingRequests sync.Map // map[requestID] chan bool
	sessionCache    []cachedPermission
	timeout         time.Duration
}

// NewService creates a permission service.
func NewService() *Service {
	return &Service{
		mode:         ModeDefault,
		sessionCache: make([]cachedPermission, 0),
		timeout:      5 * time.Minute,
	}
}

// SetMode switches the global permission mode.
func (s *Service) SetMode(mode Mode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mode = mode
}

// GetMode returns the current permission mode.
func (s *Service) GetMode() Mode {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mode
}

// RequestPermission blocks until the user responds or context expires.
// notify is called to push the SSE event (provided by the caller, e.g. agent loop).
func (s *Service) RequestPermission(ctx context.Context, req Request, notify NotifyFunc) (bool, error) {
	// Auto-approve mode: skip entirely
	if s.GetMode() == ModeAutoApprove {
		return true, nil
	}

	// Check session cache (allow_session grants)
	s.mu.RLock()
	for _, cached := range s.sessionCache {
		if cached.SessionID == req.SessionID && cached.ToolName == req.ToolName && cached.Command == req.Command {
			s.mu.RUnlock()
			return true, nil
		}
	}
	s.mu.RUnlock()

	// Generate ID if not set
	if req.ID == "" {
		req.ID = fmt.Sprintf("perm_%d", time.Now().UnixNano())
	}

	// Set expiry
	expiresAt := time.Now().Add(s.timeout)
	req.ExpiresAt = expiresAt.Format(time.RFC3339)

	// Create response channel
	respCh := make(chan bool, 1)
	s.pendingRequests.Store(req.ID, respCh)
	defer s.pendingRequests.Delete(req.ID)

	// Notify frontend via SSE
	if notify != nil {
		notify(req)
	}

	// Block until response, context cancellation, or timeout
	timeoutCtx, cancel := context.WithDeadline(ctx, expiresAt)
	defer cancel()

	select {
	case granted := <-respCh:
		return granted, nil
	case <-timeoutCtx.Done():
		return false, timeoutCtx.Err()
	}
}

// Respond is called by the HTTP handler to unblock a pending request.
func (s *Service) Respond(requestID string, action string) error {
	val, ok := s.pendingRequests.Load(requestID)
	if !ok {
		return fmt.Errorf("no pending request with ID %q", requestID)
	}

	respCh := val.(chan bool)

	switch action {
	case "allow":
		respCh <- true
	case "allow_session":
		respCh <- true
	case "deny":
		respCh <- false
	default:
		return fmt.Errorf("invalid action %q, must be allow/allow_session/deny", action)
	}

	return nil
}

// GrantForSession caches a session-level permission grant.
func (s *Service) GrantForSession(sessionID, toolName, command string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionCache = append(s.sessionCache, cachedPermission{
		SessionID: sessionID,
		ToolName:  toolName,
		Command:   command,
	})
}
