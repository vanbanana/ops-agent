// Package mcp provides MCP (Model Context Protocol) client integration.
// It connects to external MCP Servers (stdio/SSE), discovers their tools,
// and registers them into the ops-agent ToolRegistry.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"ops-agent/internal/tools"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// ServerConfig describes an external MCP Server to connect to.
type ServerConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Transport string            `json:"transport"` // "stdio" or "sse"
	Command   string            `json:"command"`   // for stdio
	Args      []string          `json:"args"`      // for stdio
	URL       string            `json:"url"`       // for sse
	Env       map[string]string `json:"env"`
	IsActive  bool              `json:"is_active"`
}

// Manager handles MCP Server connections and tool registration.
type Manager struct {
	mu       sync.RWMutex
	servers  []ServerConfig
	registry tools.ToolRegistry
}

// NewManager creates a new MCP manager.
func NewManager(registry tools.ToolRegistry) *Manager {
	return &Manager{registry: registry}
}

// ConnectAll connects to all active MCP servers and registers their tools.
func (m *Manager) ConnectAll(configs []ServerConfig) {
	m.mu.Lock()
	m.servers = configs
	m.mu.Unlock()

	for _, cfg := range configs {
		if !cfg.IsActive {
			continue
		}
		go m.connectServer(cfg)
	}
}

// connectServer connects to a single MCP server and registers its tools.
func (m *Manager) connectServer(cfg ServerConfig) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var mcpClient *client.Client
	var err error

	switch cfg.Transport {
	case "stdio":
		envSlice := make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		mcpClient, err = client.NewStdioMCPClient(cfg.Command, envSlice, cfg.Args...)
		if err != nil {
			log.Printf("[mcp] failed to create stdio client for %s: %v", cfg.Name, err)
			return
		}
	case "sse":
		mcpClient, err = client.NewSSEMCPClient(cfg.URL)
		if err != nil {
			log.Printf("[mcp] failed to create SSE client for %s: %v", cfg.Name, err)
			return
		}
	case "streamable", "http":
		tr, trErr := transport.NewStreamableHTTP(cfg.URL)
		if trErr != nil {
			log.Printf("[mcp] failed to create streamable client for %s: %v", cfg.Name, trErr)
			return
		}
		mcpClient = client.NewClient(tr)
	default:
		log.Printf("[mcp] unknown transport %q for %s", cfg.Transport, cfg.Name)
		return
	}
	defer mcpClient.Close()

	// Start transport
	if err = mcpClient.Start(ctx); err != nil {
		log.Printf("[mcp] failed to start transport for %s: %v", cfg.Name, err)
		return
	}

	// Initialize
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "ops-agent",
		Version: "1.0.0",
	}

	_, err = mcpClient.Initialize(ctx, initReq)
	if err != nil {
		log.Printf("[mcp] failed to initialize %s: %v", cfg.Name, err)
		return
	}

	// List tools
	toolsReq := mcp.ListToolsRequest{}
	toolsResult, err := mcpClient.ListTools(ctx, toolsReq)
	if err != nil {
		log.Printf("[mcp] failed to list tools from %s: %v", cfg.Name, err)
		return
	}

	// Register each tool
	for _, t := range toolsResult.Tools {
		mcpTool := &MCPTool{
			serverName: cfg.Name,
			serverCfg:  cfg,
			tool:       t,
		}
		if regErr := m.registry.Register(mcpTool); regErr != nil {
			// Tool might already be registered (duplicate name)
			log.Printf("[mcp] tool %s_%s already registered, skipping", cfg.Name, t.Name)
		}
	}

	log.Printf("[mcp] connected to %s: %d tools registered", cfg.Name, len(toolsResult.Tools))
}

// GetServers returns current server configs.
func (m *Manager) GetServers() []ServerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]ServerConfig{}, m.servers...)
}

// MCPTool wraps an MCP tool as a tools.Tool interface implementation.
type MCPTool struct {
	serverName string
	serverCfg  ServerConfig
	tool       mcp.Tool
}

func (t *MCPTool) Name() string {
	return fmt.Sprintf("%s_%s", t.serverName, t.tool.Name)
}

func (t *MCPTool) Description() string {
	return t.tool.Description
}

func (t *MCPTool) Type() tools.ToolType {
	return tools.ToolExternal
}

func (t *MCPTool) Schema() map[string]any {
	props := make(map[string]any)
	if t.tool.InputSchema.Properties != nil {
		for k, v := range t.tool.InputSchema.Properties {
			props[k] = v
		}
	}
	required := t.tool.InputSchema.Required
	if required == nil {
		required = []string{}
	}
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

func (t *MCPTool) Execute(ctx context.Context, args map[string]any) (tools.Result, error) {
	var mcpClient *client.Client
	var err error

	switch t.serverCfg.Transport {
	case "stdio":
		envSlice := make([]string, 0, len(t.serverCfg.Env))
		for k, v := range t.serverCfg.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		mcpClient, err = client.NewStdioMCPClient(t.serverCfg.Command, envSlice, t.serverCfg.Args...)
	case "sse":
		mcpClient, err = client.NewSSEMCPClient(t.serverCfg.URL)
	case "streamable", "http":
		tr, trErr := transport.NewStreamableHTTP(t.serverCfg.URL)
		if trErr != nil {
			return tools.Result{Error: fmt.Sprintf("MCP transport failed: %v", trErr)}, nil
		}
		mcpClient = client.NewClient(tr)
	default:
		return tools.Result{Error: fmt.Sprintf("unsupported transport: %s", t.serverCfg.Transport)}, nil
	}
	if err != nil {
		return tools.Result{Error: fmt.Sprintf("MCP connect failed: %v", err)}, nil
	}
	defer mcpClient.Close()

	// Start transport
	if err = mcpClient.Start(ctx); err != nil {
		return tools.Result{Error: fmt.Sprintf("MCP start failed: %v", err)}, nil
	}

	// Initialize
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "ops-agent", Version: "1.0.0"}
	if _, err = mcpClient.Initialize(ctx, initReq); err != nil {
		return tools.Result{Error: fmt.Sprintf("MCP init failed: %v", err)}, nil
	}

	// Call tool
	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = t.tool.Name
	callReq.Params.Arguments = args

	result, err := mcpClient.CallTool(ctx, callReq)
	if err != nil {
		return tools.Result{Error: fmt.Sprintf("MCP call failed: %v", err)}, nil
	}

	// Extract text content
	var output string
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text
		} else {
			b, _ := json.Marshal(content)
			output += string(b)
		}
	}

	if len(output) > 30000 {
		output = output[:30000] + "\n[truncated]"
	}

	return tools.Result{Summary: output}, nil
}
