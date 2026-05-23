// Package agent type aliases — bridges to internal/llm package.
// This allows existing code (loop.go, multi.go, tests) to continue using
// agent.Message, agent.LLMClient etc. without changing every reference.
// New code should import internal/llm directly.
package agent

import "ops-agent/internal/llm"

// Type aliases — zero cost at runtime, enables gradual migration.
type LLMClient = llm.LLMClient
type StreamChunk = llm.StreamChunk
type ChatRequest = llm.ChatRequest
type Message = llm.Message
type ToolDef = llm.ToolDef
type FunctionDef = llm.FunctionDef
type ToolCall = llm.ToolCall
type FunctionCall = llm.FunctionCall
type ChatResponse = llm.ChatResponse
type Choice = llm.Choice
type Usage = llm.Usage
type ClientConfig = llm.ClientConfig
type ErrorCode = llm.ErrorCode
type LLMError = llm.LLMError

// Re-export constants
const (
	ErrLLMNetwork = llm.ErrLLMNetwork
	ErrLLMAuth    = llm.ErrLLMAuth
	ErrLLMQuota   = llm.ErrLLMQuota
	ErrLLMService = llm.ErrLLMService
	ErrLLMTimeout = llm.ErrLLMTimeout
	ErrLLMParse   = llm.ErrLLMParse
)

// Re-export constructor
var NewLLMClient = llm.NewLLMClient
