package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// BashTool provides restricted shell command execution.
// Unlike OpenCode's fully general Bash, this is constrained to the
// safety.CommandWhitelist. The LLM uses this when no pre-built probe covers the need.
type BashTool struct{}

func NewBashTool() *BashTool { return &BashTool{} }

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Description() string {
	return `执行受限的 Shell 命令。仅允许白名单内的命令（df, du, ls, ps, ss, free, top, cat, head, tail, grep, find, journalctl, systemctl 等）。

使用场景:
- 当内置探针工具无法覆盖时，执行自定义命令获取信息
- 查看特定文件内容（cat, head, tail）
- 搜索文件内容（grep）
- 查看目录结构（ls, find）

限制:
- 命令必须在安全白名单内，否则会被拒绝
- 禁止执行网络请求命令（curl, wget）
- 禁止写操作命令（除非通过 write tools）
- 输出截断为 30000 字符
- 默认超时 30 秒

注意: 优先使用内置探针工具（probe_disk, probe_top 等），仅在探针不满足时使用此工具。`
}

func (t *BashTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "要执行的 shell 命令",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "超时秒数（默认30，最大120）",
			},
		},
		"required": []string{"command"},
	}
}

func (t *BashTool) Type() ToolType { return ToolReadOnly }

func (t *BashTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	command, _ := args["command"].(string)
	if command == "" {
		return Result{Error: "command is required"}, nil
	}

	timeoutSec := 30
	if v, ok := args["timeout"].(float64); ok && v > 0 {
		timeoutSec = int(v)
		if timeoutSec > 120 {
			timeoutSec = 120
		}
	}

	// Security: the actual validation is done by safety.ValidateCommand
	// in the agent loop before dispatch. Here we just execute.
	// But as extra safety, refuse obviously dangerous patterns.
	lower := strings.ToLower(command)
	for _, banned := range []string{"rm -rf", "mkfs", "dd if=", "> /dev/", "curl ", "wget ", "nc "} {
		if strings.Contains(lower, banned) {
			return Result{Error: fmt.Sprintf("命令被拒绝: 包含危险模式 '%s'", banned)}, nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	output, err := cmd.CombinedOutput()

	result := string(output)
	if len(result) > 30000 {
		half := 15000
		truncatedLines := strings.Count(result[half:len(result)-half], "\n")
		result = result[:half] + fmt.Sprintf("\n\n... [%d lines truncated] ...\n\n", truncatedLines) + result[len(result)-half:]
	}

	if err != nil {
		if ctx.Err() != nil {
			return Result{Summary: result, Error: "命令超时"}, nil
		}
		return Result{Summary: result, Error: fmt.Sprintf("exit: %v", err)}, nil
	}

	if result == "" {
		result = "(no output)"
	}

	return Result{Summary: result}, nil
}
