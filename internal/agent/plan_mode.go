package agent

import (
	"sync"
)

// PlanPhase represents the current phase of plan mode execution.
type PlanPhase string

const (
	PlanPhaseOff              PlanPhase = "off"               // Normal mode (no plan)
	PlanPhasePlanning         PlanPhase = "planning"          // Gathering info, read-only
	PlanPhaseAwaitingApproval PlanPhase = "awaiting_approval" // Plan presented, waiting for user
	PlanPhaseExecuting        PlanPhase = "executing"         // User approved, executing plan
	PlanPhaseCancelled        PlanPhase = "cancelled"         // User rejected
)

// PlanState manages the plan mode state machine.
// Thread-safe for concurrent access from loop + HTTP handlers.
type PlanState struct {
	mu       sync.RWMutex
	phase    PlanPhase
	planID   string
	planText string        // The plan output from LLM
	approveCh chan bool    // Channel to signal approve/reject
}

// NewPlanState creates a new plan state in OFF mode.
func NewPlanState() *PlanState {
	return &PlanState{
		phase: PlanPhaseOff,
	}
}

// Phase returns the current plan phase.
func (ps *PlanState) Phase() PlanPhase {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.phase
}

// IsPlanning returns true if we're in plan mode (gathering info).
func (ps *PlanState) IsPlanning() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.phase == PlanPhasePlanning
}

// StartPlanning enters plan mode. Only read-only tools will be allowed.
func (ps *PlanState) StartPlanning(planID string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.phase = PlanPhasePlanning
	ps.planID = planID
	ps.planText = ""
	ps.approveCh = make(chan bool, 1)
}

// SubmitPlan transitions from Planning to AwaitingApproval.
// Called when the LLM finishes its plan output.
func (ps *PlanState) SubmitPlan(planText string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.planText = planText
	ps.phase = PlanPhaseAwaitingApproval
}

// Approve transitions from AwaitingApproval to Executing.
// Called by the HTTP handler when user approves the plan.
func (ps *PlanState) Approve() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.phase == PlanPhaseAwaitingApproval {
		ps.phase = PlanPhaseExecuting
		if ps.approveCh != nil {
			ps.approveCh <- true
		}
	}
}

// Reject transitions from AwaitingApproval to Cancelled.
func (ps *PlanState) Reject() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.phase == PlanPhaseAwaitingApproval {
		ps.phase = PlanPhaseCancelled
		if ps.approveCh != nil {
			ps.approveCh <- false
		}
	}
}

// WaitForApproval blocks until the user approves or rejects.
// Returns true if approved, false if rejected.
func (ps *PlanState) WaitForApproval() <-chan bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.approveCh
}

// Reset returns to OFF mode. Used after execution completes or is cancelled.
func (ps *PlanState) Reset() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.phase = PlanPhaseOff
	ps.planID = ""
	ps.planText = ""
	ps.approveCh = nil
}

// PlanID returns the current plan identifier.
func (ps *PlanState) PlanID() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.planID
}

// PlanText returns the plan content.
func (ps *PlanState) PlanText() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.planText
}
