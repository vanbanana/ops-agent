package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type probeNetworkConnections struct{}

func NewProbeNetworkConnections() Tool { return &probeNetworkConnections{} }

func (p *probeNetworkConnections) Name() string        { return "probe_network_connections" }
func (p *probeNetworkConnections) Description() string  { return "查看网络连接和监听端口 (ss -tnlp 或 netstat)" }
func (p *probeNetworkConnections) Type() ToolType       { return ToolReadOnly }
func (p *probeNetworkConnections) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (p *probeNetworkConnections) Execute(ctx context.Context, args map[string]any) (Result, error) {
	// Try ss first (Linux), fallback to netstat (macOS/universal)
	out, err := exec.CommandContext(ctx, "ss", "-tnlp").Output()
	if err != nil {
		out, err = exec.CommandContext(ctx, "netstat", "-an").Output()
		if err != nil {
			return Result{}, fmt.Errorf("network probe failed: %w", err)
		}
	}
	summary := truncateOutput(strings.TrimSpace(string(out)), 4096)
	return Result{Summary: summary}, nil
}

type probeNetworkInterfaces struct{}

func NewProbeNetworkInterfaces() Tool { return &probeNetworkInterfaces{} }

func (p *probeNetworkInterfaces) Name() string        { return "probe_network_interfaces" }
func (p *probeNetworkInterfaces) Description() string  { return "查看网络接口信息 (ip addr 或 ifconfig)" }
func (p *probeNetworkInterfaces) Type() ToolType       { return ToolReadOnly }
func (p *probeNetworkInterfaces) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (p *probeNetworkInterfaces) Execute(ctx context.Context, args map[string]any) (Result, error) {
	out, err := exec.CommandContext(ctx, "ip", "addr").Output()
	if err != nil {
		out, err = exec.CommandContext(ctx, "ifconfig").Output()
		if err != nil {
			return Result{}, fmt.Errorf("network interfaces probe failed: %w", err)
		}
	}
	return Result{Summary: truncateOutput(strings.TrimSpace(string(out)), 4096)}, nil
}
