package tools

import "context"

// ToolType categorizes tools by their side effects.
type ToolType string

const (
	ToolReadOnly ToolType = "readonly"
	ToolWrite    ToolType = "write"
	ToolExternal ToolType = "external"
)

// Tool is the interface all tools must implement.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Type() ToolType
	Execute(ctx context.Context, args map[string]any) (Result, error)
}

// Result holds tool execution output.
type Result struct {
	Data      any    `json:"data,omitempty"`
	Summary   string `json:"summary"`
	Error     string `json:"error,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
}

// ToolDefinition is the JSON structure sent to LLM for function calling.
type ToolDefinition struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef describes a function for the LLM.
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
