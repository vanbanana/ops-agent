// Package llm provides the LLM client and types for OpenAI-compatible APIs.
// Split from internal/agent to keep agent loop focused on orchestration.
package llm

import (
	"context"
	"time"
)

// LLMClient is the interface for interacting with LLM providers.
type LLMClient interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// StreamChunk represents a single streaming delta from the LLM.
type StreamChunk struct {
	Delta          string   // text content delta
	ReasoningDelta string   // reasoning/thinking content delta (MiMo/DeepSeek)
	ToolCalls      []ToolCall // tool call deltas (accumulated)
	FinishReason   string   // set on final chunk: "stop" or "tool_calls"
	Usage          *Usage   // set on final chunk
	Done           bool     // true if this is the last chunk
	Error          error    // non-nil if streaming failed mid-way
}

// ChatRequest represents an OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Tools          []ToolDef `json:"tools,omitempty"`
	Stream         bool      `json:"stream"`
	EnableThinking bool      `json:"enable_thinking,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolDef defines a tool for function calling.
type ToolDef struct {
	Type     string      `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef is the function definition within a tool.
type FunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall represents an LLM's request to call a tool.
type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

// FunctionCall is the function name + arguments from a tool call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatResponse represents the API response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice is a single completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage tracks token consumption.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ClientConfig holds LLM connection parameters.
type ClientConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}
