package tools

import (
	"context"
	"fmt"
	"sync"
)

// ToolRegistry manages tool registration and dispatch.
type ToolRegistry interface {
	Register(t Tool) error
	Get(name string) (Tool, bool)
	List() []Tool
	Definitions() []ToolDefinition
	Dispatch(ctx context.Context, name string, args map[string]any) Result
	Enable(name string)
	Disable(name string)
	IsDisabled(name string) bool
	AllStatus() []ToolStatus
}

// ToolStatus represents a tool's registration info for the management API.
type ToolStatus struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Type        ToolType `json:"type"`
	Enabled     bool     `json:"enabled"`
	Source      string   `json:"source"` // "builtin" or "mcp"
}

// registry is the default ToolRegistry implementation.
type registry struct {
	mu       sync.RWMutex
	tools    map[string]Tool
	order    []string
	disabled map[string]bool // disabled tools are registered but not exposed to LLM
}

// NewRegistry creates a new ToolRegistry.
func NewRegistry() ToolRegistry {
	return &registry{
		tools:    make(map[string]Tool),
		disabled: make(map[string]bool),
	}
}

func (r *registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[t.Name()]; exists {
		return fmt.Errorf("tool %q already registered", t.Name())
	}
	r.tools[t.Name()] = t
	r.order = append(r.order, t.Name())
	return nil
}

func (r *registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		result = append(result, r.tools[name])
	}
	return result
}

func (r *registry) Definitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.order))
	for _, name := range r.order {
		if r.disabled[name] {
			continue
		}
		t := r.tools[name]
		defs = append(defs, ToolDefinition{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Schema(),
			},
		})
	}
	return defs
}

func (r *registry) Dispatch(ctx context.Context, name string, args map[string]any) Result {
	r.mu.RLock()
	t, ok := r.tools[name]
	isDisabled := r.disabled[name]
	r.mu.RUnlock()

	if !ok {
		return Result{Error: fmt.Sprintf("unknown tool: %s", name)}
	}
	if isDisabled {
		return Result{Error: fmt.Sprintf("tool %s is disabled", name)}
	}

	result, err := t.Execute(ctx, args)
	if err != nil {
		return Result{Error: err.Error(), Summary: "tool execution failed: " + err.Error()}
	}
	return result
}

func (r *registry) Enable(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.disabled, name)
}

func (r *registry) Disable(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tools[name]; ok {
		r.disabled[name] = true
	}
}

func (r *registry) IsDisabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.disabled[name]
}

func (r *registry) AllStatus() []ToolStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ToolStatus, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		source := "builtin"
		if t.Type() == ToolExternal {
			source = "mcp"
		}
		result = append(result, ToolStatus{
			Name:        t.Name(),
			Description: t.Description(),
			Type:        t.Type(),
			Enabled:     !r.disabled[name],
			Source:      source,
		})
	}
	return result
}
