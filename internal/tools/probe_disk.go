package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type probeDisk struct{}

func NewProbeDisk() Tool { return &probeDisk{} }

func (p *probeDisk) Name() string        { return "probe_disk" }
func (p *probeDisk) Description() string  { return "查看磁盘使用情况，返回各分区的使用率 (df -h)" }
func (p *probeDisk) Type() ToolType       { return ToolReadOnly }
func (p *probeDisk) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "要查看的路径，默认 /"},
		},
	}
}

func (p *probeDisk) Execute(ctx context.Context, args map[string]any) (Result, error) {
	out, err := exec.CommandContext(ctx, "df", "-h").Output()
	if err != nil {
		return Result{}, fmt.Errorf("df -h failed: %w", err)
	}
	summary := string(out)
	return Result{Summary: strings.TrimSpace(summary), Data: summary}, nil
}

type probeLargeFiles struct{}

func NewProbeLargeFiles() Tool { return &probeLargeFiles{} }

func (p *probeLargeFiles) Name() string        { return "probe_large_files" }
func (p *probeLargeFiles) Description() string  { return "查找指定路径下的大文件 (find -size)" }
func (p *probeLargeFiles) Type() ToolType       { return ToolReadOnly }
func (p *probeLargeFiles) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":     map[string]any{"type": "string", "description": "搜索路径，默认 /var/log"},
			"min_size": map[string]any{"type": "string", "description": "最小文件大小，默认 100M"},
		},
	}
}

func (p *probeLargeFiles) Execute(ctx context.Context, args map[string]any) (Result, error) {
	path := getStringArg(args, "path", "/var/log")
	minSize := getStringArg(args, "min_size", "100M")

	// Use -maxdepth 4 to prevent find from running indefinitely on deep/circular paths
	// Use a 15-second timeout context to fail fast instead of blocking the loop
	findCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	out, err := exec.CommandContext(findCtx, "find", path, "-maxdepth", "4", "-type", "f", "-size", "+"+minSize).Output()
	if err != nil {
		if findCtx.Err() != nil {
			return Result{Summary: "搜索超时（路径过深），建议缩小搜索范围或指定具体子目录"}, nil
		}
		// find returns exit 1 for permission denied on some dirs, that's ok
		if len(out) > 0 {
			return Result{Summary: strings.TrimSpace(string(out))}, nil
		}
		return Result{Summary: "未找到大于 " + minSize + " 的文件或路径不可访问"}, nil
	}
	summary := strings.TrimSpace(string(out))
	if summary == "" {
		summary = "未找到大于 " + minSize + " 的文件"
	}
	return Result{Summary: truncateOutput(summary, 4096)}, nil
}

func getStringArg(args map[string]any, key, defaultVal string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultVal
}

func truncateOutput(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "\n\n[已截断, 共 " + fmt.Sprintf("%d", len(s)) + " 字符]"
}
