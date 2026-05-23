package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type probeLogsJournal struct{}

func NewProbeLogsJournal() Tool { return &probeLogsJournal{} }

func (p *probeLogsJournal) Name() string        { return "probe_logs_journal" }
func (p *probeLogsJournal) Description() string  { return "查看系统日志 (journalctl --since)" }
func (p *probeLogsJournal) Type() ToolType       { return ToolReadOnly }
func (p *probeLogsJournal) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"since": map[string]any{"type": "string", "description": "时间范围，如 '1 hour ago'，默认 '30 min ago'"},
			"unit":  map[string]any{"type": "string", "description": "服务名，如 nginx"},
			"lines": map[string]any{"type": "integer", "description": "返回行数，默认 50"},
		},
	}
}

func (p *probeLogsJournal) Execute(ctx context.Context, args map[string]any) (Result, error) {
	since := getStringArg(args, "since", "30 min ago")
	cmdArgs := []string{"--since", since, "--no-pager", "-n", "50"}

	if unit := getStringArg(args, "unit", ""); unit != "" {
		cmdArgs = append(cmdArgs, "-u", unit)
	}

	out, err := exec.CommandContext(ctx, "journalctl", cmdArgs...).Output()
	if err != nil {
		return Result{Summary: "journalctl 不可用（macOS 无 systemd 或权限不足）"}, nil
	}
	return Result{Summary: truncateOutput(strings.TrimSpace(string(out)), 4096)}, nil
}

type probeLogsFile struct{}

func NewProbeLogsFile() Tool { return &probeLogsFile{} }

func (p *probeLogsFile) Name() string        { return "probe_logs_file" }
func (p *probeLogsFile) Description() string  { return "查看指定日志文件末尾内容 (tail -n)" }
func (p *probeLogsFile) Type() ToolType       { return ToolReadOnly }
func (p *probeLogsFile) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":  map[string]any{"type": "string", "description": "日志文件路径"},
			"lines": map[string]any{"type": "integer", "description": "返回行数，默认 50"},
		},
		"required": []string{"path"},
	}
}

func (p *probeLogsFile) Execute(ctx context.Context, args map[string]any) (Result, error) {
	path := getStringArg(args, "path", "")
	if path == "" {
		return Result{Error: "path is required"}, fmt.Errorf("path is required")
	}

	lines := "50"
	if v, ok := args["lines"]; ok {
		switch n := v.(type) {
		case float64:
			lines = fmt.Sprintf("%d", int(n))
		case string:
			lines = n
		}
	}

	out, err := exec.CommandContext(ctx, "tail", "-n", lines, path).Output()
	if err != nil {
		return Result{}, fmt.Errorf("tail %s failed: %w", path, err)
	}
	return Result{Summary: truncateOutput(strings.TrimSpace(string(out)), 4096)}, nil
}
