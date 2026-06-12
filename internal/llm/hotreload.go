package llm

import (
	"context"
	"sync"
	"time"
)

// HotReloadClient wraps an LLMClient with ability to swap config at runtime.
type HotReloadClient struct {
	mu     sync.RWMutex
	client LLMClient
	config ClientConfig
}

// NewHotReloadClient creates a hot-reloadable LLM client wrapper.
func NewHotReloadClient(cfg ClientConfig) *HotReloadClient {
	return &HotReloadClient{
		client: NewLLMClient(cfg),
		config: cfg,
	}
}

// Reload swaps the underlying client config (base_url, api_key, model).
func (h *HotReloadClient) Reload(cfg ClientConfig) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.config = cfg
	h.client = NewLLMClient(cfg)
}

// Config returns the current client config (safe for reading model name etc).
func (h *HotReloadClient) Config() ClientConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.config
}

// Chat delegates to the current underlying client.
func (h *HotReloadClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	h.mu.RLock()
	c := h.client
	h.mu.RUnlock()
	return c.Chat(ctx, req)
}

// ChatStream delegates to the current underlying client.
func (h *HotReloadClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	h.mu.RLock()
	c := h.client
	h.mu.RUnlock()
	return c.ChatStream(ctx, req)
}
