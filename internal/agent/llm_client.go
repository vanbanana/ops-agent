package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClient interface — design.md §2.3
type LLMClient interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// StreamChunk represents a single streaming delta from the LLM.
type StreamChunk struct {
	Delta            string     // text content delta
	ReasoningDelta   string     // reasoning/thinking content delta (MiMo/DeepSeek)
	ToolCalls        []ToolCall // tool call deltas (accumulated)
	FinishReason     string     // set on final chunk: "stop" or "tool_calls"
	Usage            *Usage     // set on final chunk
	Done             bool       // true if this is the last chunk
	Error            error      // non-nil if streaming failed mid-way
}

// ChatRequest represents an OpenAI-compatible chat completion request.
type ChatRequest struct {
	Model          string        `json:"model"`
	Messages       []Message     `json:"messages"`
	Tools          []ToolDef     `json:"tools,omitempty"`
	Stream         bool          `json:"stream"`
	EnableThinking bool          `json:"enable_thinking,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ToolDef defines a tool for function calling.
type ToolDef struct {
	Type     string       `json:"type"`
	Function FunctionDef  `json:"function"`
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

// httpLLMClient implements LLMClient via OpenAI-compatible HTTP API.
type httpLLMClient struct {
	config ClientConfig
	client *http.Client
}

// NewLLMClient creates a new LLM client with the given config.
func NewLLMClient(cfg ClientConfig) LLMClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	return &httpLLMClient{
		config: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Chat sends a chat completion request and returns the response.
func (c *httpLLMClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Model = c.config.Model
	req.Stream = false

	body, err := json.Marshal(req)
	if err != nil {
		return nil, &LLMError{Code: ErrLLMParse, Message: "marshal request", Err: err}
	}

	url := c.config.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, &LLMError{Code: ErrLLMNetwork, Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		// Distinguish timeout from general network errors
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &LLMError{Code: ErrLLMTimeout, Message: "request timed out", Err: err}
		}
		return nil, &LLMError{Code: ErrLLMNetwork, Message: "request failed", Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &LLMError{Code: ErrLLMNetwork, Message: "read response", Err: err}
	}

	// Handle HTTP error codes
	if resp.StatusCode != http.StatusOK {
		return nil, classifyHTTPError(resp.StatusCode, respBody)
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, &LLMError{Code: ErrLLMParse, Message: fmt.Sprintf("unmarshal response: %s", string(respBody)), Err: err}
	}

	return &chatResp, nil
}

// classifyHTTPError maps HTTP status codes to typed LLM errors.
func classifyHTTPError(status int, body []byte) *LLMError {
	msg := string(body)
	switch {
	case status == 401 || status == 403:
		return &LLMError{Code: ErrLLMAuth, Message: fmt.Sprintf("HTTP %d: %s", status, msg)}
	case status == 429:
		return &LLMError{Code: ErrLLMQuota, Message: fmt.Sprintf("HTTP %d: rate limited", status)}
	case status >= 500:
		return &LLMError{Code: ErrLLMService, Message: fmt.Sprintf("HTTP %d: %s", status, msg)}
	default:
		return &LLMError{Code: ErrLLMService, Message: fmt.Sprintf("HTTP %d: %s", status, msg)}
	}
}

// ChatStream sends a streaming chat completion request (OpenAI-compatible protocol).
// Works with any provider that supports /v1/chat/completions with stream:true
// (DeepSeek, MiMo, 通义千问, MiniMax, OpenAI, etc.)
// Includes automatic retry with exponential backoff for 429/500 errors.
func (c *httpLLMClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	req.Model = c.config.Model
	req.Stream = true

	body, err := json.Marshal(req)
	if err != nil {
		return nil, &LLMError{Code: ErrLLMParse, Message: "marshal request", Err: err}
	}

	ch := make(chan StreamChunk, 64)

	go func() {
		defer close(ch)

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := c.doStream(ctx, body, ch)
			if err == nil {
				return // success
			}

			// Check if retryable
			if !isRetryableError(err) || attempt >= maxRetries {
				ch <- StreamChunk{Error: err, Done: true}
				return
			}

			// Exponential backoff: 2^attempt * 500ms + jitter
			backoff := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
			jitter := time.Duration(attempt*137) * time.Millisecond // deterministic jitter
			wait := backoff + jitter
			if wait > 10*time.Second {
				wait = 10 * time.Second
			}

			select {
			case <-ctx.Done():
				ch <- StreamChunk{Error: ctx.Err(), Done: true}
				return
			case <-time.After(wait):
				continue
			}
		}
	}()

	return ch, nil
}

const maxRetries = 5

// doStream performs a single streaming request attempt. Returns nil on success (data written to ch).
// Returns error if the request failed and should potentially be retried.
func (c *httpLLMClient) doStream(ctx context.Context, body []byte, ch chan<- StreamChunk) error {
	url := c.config.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return &LLMError{Code: ErrLLMNetwork, Message: "create request", Err: err}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &LLMError{Code: ErrLLMTimeout, Message: "request timed out", Err: err}
		}
		return &LLMError{Code: ErrLLMNetwork, Message: "request failed", Err: err}
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return classifyHTTPError(resp.StatusCode, respBody)
	}

	// Stream response body
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	var leftover string
	var accumulatedToolCalls []ToolCall

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			leftover += string(buf[:n])
			for {
				idx := indexOf(leftover, "\n")
				if idx < 0 {
					break
				}
				line := leftover[:idx]
				leftover = leftover[idx+1:]

				if len(line) == 0 || line == "\r" {
					continue
				}
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}

				if line == "data: [DONE]" {
					ch <- StreamChunk{Done: true}
					return nil
				}

				if len(line) > 6 && line[:6] == "data: " {
					data := line[6:]
					var chunk streamResponseChunk
					if jsonErr := json.Unmarshal([]byte(data), &chunk); jsonErr != nil {
						continue
					}
					if len(chunk.Choices) == 0 {
						if chunk.Usage.TotalTokens > 0 {
							ch <- StreamChunk{Usage: &Usage{
								PromptTokens:     chunk.Usage.PromptTokens,
								CompletionTokens: chunk.Usage.CompletionTokens,
								TotalTokens:      chunk.Usage.TotalTokens,
							}}
						}
						continue
					}
					delta := chunk.Choices[0].Delta
					finish := chunk.Choices[0].FinishReason

					for _, tc := range delta.ToolCalls {
						for len(accumulatedToolCalls) <= tc.Index {
							accumulatedToolCalls = append(accumulatedToolCalls, ToolCall{})
						}
						if tc.ID != "" {
							accumulatedToolCalls[tc.Index].ID = tc.ID
							accumulatedToolCalls[tc.Index].Type = "function"
						}
						if tc.Function.Name != "" {
							accumulatedToolCalls[tc.Index].Function.Name = tc.Function.Name
						}
						accumulatedToolCalls[tc.Index].Function.Arguments += tc.Function.Arguments
					}

					sc := StreamChunk{Delta: delta.Content, ReasoningDelta: delta.ReasoningContent}
					if finish == "stop" || finish == "tool_calls" {
						sc.FinishReason = finish
						sc.Done = true
						if finish == "tool_calls" {
							sc.ToolCalls = accumulatedToolCalls
						}
						if chunk.Usage.TotalTokens > 0 {
							sc.Usage = &Usage{
								PromptTokens:     chunk.Usage.PromptTokens,
								CompletionTokens: chunk.Usage.CompletionTokens,
								TotalTokens:      chunk.Usage.TotalTokens,
							}
						}
					}
					ch <- sc
					if sc.Done {
						return nil
					}
				}
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				return &LLMError{Code: ErrLLMNetwork, Message: "stream read error", Err: readErr}
			}
			ch <- StreamChunk{Done: true}
			return nil
		}
	}
}

// isRetryableError checks if an LLM error is retryable (429 rate limit or 500 server error).
func isRetryableError(err error) bool {
	if llmErr, ok := err.(*LLMError); ok {
		return llmErr.Code == ErrLLMQuota || llmErr.Code == ErrLLMService
	}
	return false
}

// streamResponseChunk is the SSE JSON structure for streaming responses (OpenAI-compatible).
type streamResponseChunk struct {
	ID      string `json:"id"`
	Choices []struct {
		Index        int    `json:"index"`
		Delta        streamDelta `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

type streamDelta struct {
	Role             string            `json:"role,omitempty"`
	Content          string            `json:"content,omitempty"`
	ReasoningContent string            `json:"reasoning_content,omitempty"`
	ToolCalls        []streamToolCall  `json:"tool_calls,omitempty"`
}

type streamToolCall struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function"`
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
