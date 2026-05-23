package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type probeProcess struct{}

func NewProbeProcess() Tool { return &probeProcess{} }

func (p *probeProcess) Name() string        { return "probe_process" }
func (p *probeProcess) Description() string  { return "查看进程列表，返回 CPU/内存占用 top10 (ps aux)" }
func (p *probeProcess) Type() ToolType       { return ToolReadOnly }
func (p *probeProcess) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (p *probeProcess) Execute(ctx context.Context, args map[string]any) (Result, error) {
	out, err := exec.CommandContext(ctx, "ps", "aux").Output()
	if err != nil {
		return Result{}, fmt.Errorf("ps aux failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	total := len(lines) - 1 // subtract header
	if total < 0 {
		total = 0
	}

	// Keep header + top 10 lines (sorted by default ps output)
	var summary string
	if len(lines) <= 11 {
		summary = string(out)
	} else {
		top := append([]string{lines[0]}, lines[1:11]...)
		summary = strings.Join(top, "\n") + fmt.Sprintf("\n\n[共 %d 个进程, 仅显示前 10 个]", total)
	}

	return Result{
		Summary:   summary,
		Data:      map[string]any{"total_processes": total},
		Truncated: len(lines) > 11,
	}, nil
}

type probeTop struct{}

func NewProbeTop() Tool { return &probeTop{} }

func (p *probeTop) Name() string        { return "probe_top" }
func (p *probeTop) Description() string  { return "查看系统负载概览 (top -bn1 或 等效)" }
func (p *probeTop) Type() ToolType       { return ToolReadOnly }
func (p *probeTop) Schema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (p *probeTop) Execute(ctx context.Context, args map[string]any) (Result, error) {
	// macOS uses different top flags than Linux
	out, err := exec.CommandContext(ctx, "top", "-l", "1", "-n", "10").Output()
	if err != nil {
		// Try Linux style
		out, err = exec.CommandContext(ctx, "top", "-bn1").Output()
		if err != nil {
			return Result{}, fmt.Errorf("top failed: %w", err)
		}
	}

	lines := strings.Split(string(out), "\n")
	// Keep first 20 lines as summary
	if len(lines) > 20 {
		lines = lines[:20]
	}
	return Result{Summary: strings.Join(lines, "\n")}, nil
}
