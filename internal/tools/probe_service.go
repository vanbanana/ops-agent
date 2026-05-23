package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type probeServiceStatus struct{}

func NewProbeServiceStatus() Tool { return &probeServiceStatus{} }

func (p *probeServiceStatus) Name() string        { return "probe_service_status" }
func (p *probeServiceStatus) Description() string  { return "查看系统服务状态 (systemctl status)" }
func (p *probeServiceStatus) Type() ToolType       { return ToolReadOnly }
func (p *probeServiceStatus) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"service": map[string]any{"type": "string", "description": "服务名，如 nginx, mysql"},
		},
		"required": []string{"service"},
	}
}

func (p *probeServiceStatus) Execute(ctx context.Context, args map[string]any) (Result, error) {
	service := getStringArg(args, "service", "")
	if service == "" {
		return Result{Error: "service name is required"}, fmt.Errorf("service is required")
	}

	out, err := exec.CommandContext(ctx, "systemctl", "status", service).Output()
	if err != nil {
		// systemctl returns exit 3 for inactive services, still has output
		if len(out) > 0 {
			return Result{Summary: strings.TrimSpace(string(out))}, nil
		}
		return Result{Summary: fmt.Sprintf("服务 %s 状态查询失败（可能不在 systemd 环境）", service)}, nil
	}
	return Result{Summary: strings.TrimSpace(string(out))}, nil
}

type probeFileHolders struct{}

func NewProbeFileHolders() Tool { return &probeFileHolders{} }

func (p *probeFileHolders) Name() string        { return "probe_file_holders" }
func (p *probeFileHolders) Description() string  { return "查看文件被哪些进程打开 (lsof)" }
func (p *probeFileHolders) Type() ToolType       { return ToolReadOnly }
func (p *probeFileHolders) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "文件路径或端口(如 :80)"},
		},
		"required": []string{"path"},
	}
}

func (p *probeFileHolders) Execute(ctx context.Context, args map[string]any) (Result, error) {
	path := getStringArg(args, "path", "")
	if path == "" {
		return Result{Error: "path is required"}, fmt.Errorf("path is required")
	}

	var cmdArgs []string
	if strings.HasPrefix(path, ":") {
		cmdArgs = []string{"-i", path}
	} else {
		cmdArgs = []string{path}
	}

	out, err := exec.CommandContext(ctx, "lsof", cmdArgs...).Output()
	if err != nil {
		if len(out) > 0 {
			return Result{Summary: strings.TrimSpace(string(out))}, nil
		}
		return Result{Summary: "未找到打开该文件/端口的进程"}, nil
	}
	return Result{Summary: truncateOutput(strings.TrimSpace(string(out)), 4096)}, nil
}
