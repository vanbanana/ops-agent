package llm

// ModelInfo describes a supported LLM model's capabilities and limits.
type ModelInfo struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ContextWindow int    `json:"context_window"` // max tokens
	MaxOutput     int    `json:"max_output"`     // max output tokens
	CanReason     bool   `json:"can_reason"`     // supports reasoning_content
	CanThink      bool   `json:"can_think"`      // supports enable_thinking param
}

// SupportedModels is the registry of known models with their context windows.
// Used by compaction logic to know when to compress.
var SupportedModels = map[string]ModelInfo{
	"mimo-v2.5-pro": {
		ID: "mimo-v2.5-pro", Name: "MiMo V2.5 Pro",
		ContextWindow: 32768, MaxOutput: 8192,
		CanReason: true, CanThink: true,
	},
	"mimo-v2-flash": {
		ID: "mimo-v2-flash", Name: "MiMo V2 Flash",
		ContextWindow: 256000, MaxOutput: 16384,
		CanReason: true, CanThink: true,
	},
	"deepseek-v4-flash": {
		ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash",
		ContextWindow: 128000, MaxOutput: 8192,
		CanReason: true, CanThink: true,
	},
	"deepseek-v4-pro": {
		ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro",
		ContextWindow: 128000, MaxOutput: 8192,
		CanReason: true, CanThink: true,
	},
	"qwen3.6-plus": {
		ID: "qwen3.6-plus", Name: "Qwen 3.6 Plus",
		ContextWindow: 131072, MaxOutput: 8192,
		CanReason: true, CanThink: true,
	},
}

// GetModelInfo returns info for a model ID. Falls back to a conservative default.
func GetModelInfo(modelID string) ModelInfo {
	if info, ok := SupportedModels[modelID]; ok {
		return info
	}
	// Conservative default for unknown models
	return ModelInfo{
		ID: modelID, Name: modelID,
		ContextWindow: 16384, MaxOutput: 4096,
		CanReason: false, CanThink: false,
	}
}

// ContextBudget returns the token budget for message history (80% of context window minus output reserve).
func (m ModelInfo) ContextBudget() int {
	return int(float64(m.ContextWindow-m.MaxOutput) * 0.8)
}
