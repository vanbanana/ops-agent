package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// --- 6 Controlled Write Tools (Task 12.2) ---
// All write tools require risk preview confirmation before actual execution.
// They are registered with ToolWrite type so the safety layer can intercept them.

// ServiceControlTool — systemctl start/stop/restart a unit
type ServiceControlTool struct{}

func (t *ServiceControlTool) Name() string        { return "service_control" }
func (t *ServiceControlTool) Description() string { return "启停服务: systemctl start|stop|restart <unit>" }
func (t *ServiceControlTool) Type() ToolType      { return ToolWrite }
func (t *ServiceControlTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{"type": "string", "enum": []string{"start", "stop", "restart"}, "description": "操作类型"},
			"unit":   map[string]any{"type": "string", "description": "systemd unit 名称，如 nginx.service"},
		},
		"required": []string{"action", "unit"},
	}
}

var unitNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_@.-]+\.service$`)

func (t *ServiceControlTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	action, _ := args["action"].(string)
	unit, _ := args["unit"].(string)

	if !unitNameRegex.MatchString(unit) {
		return Result{Error: fmt.Sprintf("invalid unit name format: %q (must match xxx.service)", unit)}, nil
	}
	if action != "start" && action != "stop" && action != "restart" {
		return Result{Error: fmt.Sprintf("invalid action: %q", action)}, nil
	}

	cmd := exec.CommandContext(ctx, "systemctl", action, unit)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Error: fmt.Sprintf("systemctl %s %s failed: %v\n%s", action, unit, err, string(out))}, nil
	}
	return Result{Data: string(out), Summary: fmt.Sprintf("systemctl %s %s 成功", action, unit)}, nil
}

// TruncateLogTool — truncate a log file to 0 bytes
type TruncateLogTool struct{}

func (t *TruncateLogTool) Name() string        { return "truncate_log_file" }
func (t *TruncateLogTool) Description() string { return "清空日志文件 (截断为0字节)" }
func (t *TruncateLogTool) Type() ToolType      { return ToolWrite }
func (t *TruncateLogTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "日志文件路径，必须在 /var/log/ 下"},
		},
		"required": []string{"path"},
	}
}

func (t *TruncateLogTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	path, _ := args["path"].(string)
	cleaned := filepath.Clean(path)

	if !strings.HasPrefix(cleaned, "/var/log/") {
		return Result{Error: fmt.Sprintf("路径 %q 不在 /var/log/ 下，拒绝操作", cleaned)}, nil
	}
	if strings.Contains(cleaned, "..") {
		return Result{Error: "路径包含 .. ，拒绝操作"}, nil
	}

	cmd := exec.CommandContext(ctx, "truncate", "-s", "0", cleaned)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Error: fmt.Sprintf("truncate failed: %v\n%s", err, string(out))}, nil
	}
	return Result{Data: "ok", Summary: fmt.Sprintf("已清空 %s", cleaned)}, nil
}

// DeleteFileTool — rm a specific file (not directory)
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string        { return "delete_file" }
func (t *DeleteFileTool) Description() string { return "删除指定文件 (非目录)" }
func (t *DeleteFileTool) Type() ToolType      { return ToolWrite }
func (t *DeleteFileTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "要删除的文件路径"},
		},
		"required": []string{"path"},
	}
}

func (t *DeleteFileTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	path, _ := args["path"].(string)
	cleaned := filepath.Clean(path)

	// Safety: only allow in /tmp, /var/log, /var/tmp
	allowed := false
	for _, prefix := range []string{"/tmp/", "/var/log/", "/var/tmp/"} {
		if strings.HasPrefix(cleaned, prefix) {
			allowed = true
			break
		}
	}
	if !allowed {
		return Result{Error: fmt.Sprintf("路径 %q 不在允许删除范围内 (/tmp, /var/log, /var/tmp)", cleaned)}, nil
	}

	cmd := exec.CommandContext(ctx, "rm", "-f", cleaned)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Error: fmt.Sprintf("rm failed: %v\n%s", err, string(out))}, nil
	}
	return Result{Data: "ok", Summary: fmt.Sprintf("已删除 %s", cleaned)}, nil
}

// VacuumJournalTool — journalctl --vacuum-size
type VacuumJournalTool struct{}

func (t *VacuumJournalTool) Name() string        { return "vacuum_journal" }
func (t *VacuumJournalTool) Description() string { return "清理 systemd journal 日志 (--vacuum-size)" }
func (t *VacuumJournalTool) Type() ToolType      { return ToolWrite }
func (t *VacuumJournalTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"max_size": map[string]any{"type": "string", "description": "保留最大大小，如 500M 或 1G"},
		},
		"required": []string{"max_size"},
	}
}

func (t *VacuumJournalTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	maxSize, _ := args["max_size"].(string)
	if maxSize == "" {
		maxSize = "500M"
	}
	// Validate format: digits + M/G
	if !regexp.MustCompile(`^\d+[MG]$`).MatchString(maxSize) {
		return Result{Error: fmt.Sprintf("invalid max_size format: %q (use e.g. 500M or 1G)", maxSize)}, nil
	}

	cmd := exec.CommandContext(ctx, "journalctl", "--vacuum-size="+maxSize)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Error: fmt.Sprintf("journalctl vacuum failed: %v\n%s", err, string(out))}, nil
	}
	return Result{Data: string(out), Summary: fmt.Sprintf("journal 已清理至 %s", maxSize)}, nil
}

// LogrotateTool — force logrotate
type LogrotateTool struct{}

func (t *LogrotateTool) Name() string        { return "logrotate_now" }
func (t *LogrotateTool) Description() string { return "立即执行 logrotate 日志轮转" }
func (t *LogrotateTool) Type() ToolType      { return ToolWrite }
func (t *LogrotateTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"config": map[string]any{"type": "string", "description": "logrotate 配置文件路径，默认 /etc/logrotate.conf"},
		},
	}
}

func (t *LogrotateTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	config, _ := args["config"].(string)
	if config == "" {
		config = "/etc/logrotate.conf"
	}
	config = filepath.Clean(config)
	if !strings.HasPrefix(config, "/etc/") {
		return Result{Error: "logrotate 配置文件必须在 /etc/ 下"}, nil
	}

	cmd := exec.CommandContext(ctx, "logrotate", "-f", config)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Error: fmt.Sprintf("logrotate failed: %v\n%s", err, string(out))}, nil
	}
	return Result{Data: string(out), Summary: "logrotate 执行完成"}, nil
}

// KillProcessTool — kill a process by PID (SIGTERM only, no SIGKILL)
type KillProcessTool struct{}

func (t *KillProcessTool) Name() string        { return "kill_process" }
func (t *KillProcessTool) Description() string { return "终止进程 (仅 SIGTERM，禁止 SIGKILL)" }
func (t *KillProcessTool) Type() ToolType      { return ToolWrite }
func (t *KillProcessTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pid":    map[string]any{"type": "integer", "description": "进程 PID"},
			"signal": map[string]any{"type": "string", "description": "信号名称 (仅允许 TERM/HUP/USR1/USR2)", "default": "TERM"},
		},
		"required": []string{"pid"},
	}
}

var allowedSignals = map[string]bool{"TERM": true, "HUP": true, "USR1": true, "USR2": true}

func (t *KillProcessTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	pidRaw, _ := args["pid"]
	signal, _ := args["signal"].(string)
	if signal == "" {
		signal = "TERM"
	}
	signal = strings.ToUpper(signal)

	// Parse PID
	var pid int
	switch v := pidRaw.(type) {
	case float64:
		pid = int(v)
	case string:
		var err error
		pid, err = strconv.Atoi(v)
		if err != nil {
			return Result{Error: fmt.Sprintf("invalid pid: %q", v)}, nil
		}
	default:
		return Result{Error: "pid must be a number"}, nil
	}

	if pid <= 1 {
		return Result{Error: "拒绝 kill PID ≤ 1 (init/systemd)"}, nil
	}

	if !allowedSignals[signal] {
		return Result{Error: fmt.Sprintf("信号 %q 不允许，仅支持: TERM, HUP, USR1, USR2（禁止 KILL/9）", signal)}, nil
	}

	cmd := exec.CommandContext(ctx, "kill", fmt.Sprintf("-%s", signal), strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Result{Error: fmt.Sprintf("kill -%s %d failed: %v\n%s", signal, pid, err, string(out))}, nil
	}
	return Result{Data: "ok", Summary: fmt.Sprintf("已发送 SIG%s 到 PID %d", signal, pid)}, nil
}
