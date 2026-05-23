package agent

// Mode determines single vs multi agent routing.
type Mode string

const (
	ModeSingle Mode = "single"
	ModeMulti  Mode = "multi"
	ModeAuto   Mode = "auto"
)

// ComplexityDecision holds the routing decision and reason.
type ComplexityDecision struct {
	Mode   Mode   `json:"mode"`
	Reason string `json:"reason"`
}

// ComplexityJudge is retained for API compatibility.
// Unlike the previous LLM-backed implementation, this returns instantly.
// Routing to multi-agent is now handled by the LLM itself via the
// multi_agent_analyze tool (OpenCode pattern: model-as-router).
type ComplexityJudge struct{}

// NewComplexityJudge creates a judge. No LLM dependency needed.
func NewComplexityJudge(_ LLMClient) *ComplexityJudge {
	return &ComplexityJudge{}
}

// Decide always returns single mode instantly (zero latency).
// Multi-agent is triggered by the LLM choosing to call multi_agent_analyze tool.
// The only exception is explicit force mode from the API caller.
func (j *ComplexityJudge) Decide(_ interface{}, message string, forceMode Mode) ComplexityDecision {
	if forceMode == ModeMulti {
		return ComplexityDecision{Mode: ModeMulti, Reason: "强制多Agent模式"}
	}
	return ComplexityDecision{Mode: ModeSingle, Reason: "default"}
}
