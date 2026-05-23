package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type probeMemory struct{}

func NewProbeMemory() Tool { return &probeMemory{} }

func (p *probeMemory) Name() string        { return "probe_memory" }
func (p *probeMemory) Description() string  { return "查看内存使用情况 (free -h)" }
func (p *probeMemory) Type() ToolType       { return ToolReadOnly }
func (p *probeMemory) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (p *probeMemory) Execute(ctx context.Context, args map[string]any) (Result, error) {
	out, err := exec.CommandContext(ctx, "free", "-h").Output()
	if err != nil {
		// macOS doesn't have free, use vm_stat
		out, err = exec.CommandContext(ctx, "vm_stat").Output()
		if err != nil {
			return Result{}, fmt.Errorf("memory probe failed: %w", err)
		}
	}
	return Result{Summary: strings.TrimSpace(string(out))}, nil
}

type probeSystemInfo struct{}

func NewProbeSystemInfo() Tool { return &probeSystemInfo{} }

func (p *probeSystemInfo) Name() string        { return "probe_system_info" }
func (p *probeSystemInfo) Description() string  { return "查看系统基本信息 (uname -a + uptime)" }
func (p *probeSystemInfo) Type() ToolType       { return ToolReadOnly }
func (p *probeSystemInfo) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (p *probeSystemInfo) Execute(ctx context.Context, args map[string]any) (Result, error) {
	var parts []string

	if out, err := exec.CommandContext(ctx, "uname", "-a").Output(); err == nil {
		parts = append(parts, "System: "+strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, "uptime").Output(); err == nil {
		parts = append(parts, "Uptime: "+strings.TrimSpace(string(out)))
	}

	if len(parts) == 0 {
		return Result{}, fmt.Errorf("system info probe failed")
	}
	return Result{Summary: strings.Join(parts, "\n")}, nil
}
