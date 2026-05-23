package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileViewTool reads and displays file contents with line numbers.
// Mirrors OpenCode's View tool but scoped to ops/config file reading.
type FileViewTool struct{}

func NewFileViewTool() *FileViewTool { return &FileViewTool{} }

func (t *FileViewTool) Name() string { return "file_view" }

func (t *FileViewTool) Description() string {
	return `查看文件内容。读取指定文件并显示带行号的内容。

使用场景:
- 查看配置文件（nginx.conf, my.cnf, sshd_config 等）
- 查看日志文件的特定部分
- 检查脚本内容
- 查看 crontab、systemd unit 文件等

限制:
- 最大文件 250KB
- 默认读取前 200 行（可通过 offset + limit 分页）
- 禁止读取敏感文件（/etc/shadow, SSH 私钥等）
- 每行超过 500 字符会截断`
}

func (t *FileViewTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "文件路径（绝对路径或相对于工作目录）",
			},
			"offset": map[string]any{
				"type":        "integer",
				"description": "起始行号（0-based，默认0）",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "读取行数（默认200，最大2000）",
			},
		},
		"required": []string{"path"},
	}
}

func (t *FileViewTool) Type() ToolType { return ToolReadOnly }

const (
	maxFileSize    = 250 * 1024 // 250KB
	defaultLimit   = 200
	maxLimit       = 2000
	maxLineLength  = 500
)

// Sensitive files that should never be read
var sensitiveFiles = map[string]bool{
	"/etc/shadow":                true,
	"/etc/gshadow":              true,
	"/etc/sudoers":              true,
	"/root/.ssh/id_rsa":         true,
	"/root/.ssh/id_ed25519":     true,
	"/root/.bash_history":       true,
}

func (t *FileViewTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	filePath, _ := args["path"].(string)
	if filePath == "" {
		return Result{Error: "path is required"}, nil
	}

	// Handle relative paths
	if !filepath.IsAbs(filePath) {
		cwd, _ := os.Getwd()
		filePath = filepath.Join(cwd, filePath)
	}
	filePath = filepath.Clean(filePath)

	// Security: block sensitive files
	if sensitiveFiles[filePath] {
		return Result{Error: fmt.Sprintf("禁止访问敏感文件: %s", filePath)}, nil
	}
	// Block SSH keys by pattern
	if strings.Contains(filePath, "/.ssh/id_") || strings.HasSuffix(filePath, ".pem") || strings.HasSuffix(filePath, ".key") {
		return Result{Error: fmt.Sprintf("禁止访问密钥文件: %s", filePath)}, nil
	}

	// Check file exists and size
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{Error: fmt.Sprintf("文件不存在: %s", filePath)}, nil
		}
		return Result{Error: fmt.Sprintf("无法访问文件: %v", err)}, nil
	}
	if info.IsDir() {
		return Result{Error: fmt.Sprintf("路径是目录不是文件: %s", filePath)}, nil
	}
	if info.Size() > maxFileSize {
		return Result{Error: fmt.Sprintf("文件过大 (%d bytes)，最大 %d bytes", info.Size(), maxFileSize)}, nil
	}

	// Parse offset and limit
	offset := 0
	if v, ok := args["offset"].(float64); ok && v > 0 {
		offset = int(v)
	}
	limit := defaultLimit
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return Result{Error: fmt.Sprintf("读取失败: %v", err)}, nil
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	// Apply offset
	if offset >= totalLines {
		return Result{Summary: fmt.Sprintf("(文件共 %d 行，offset %d 超出范围)", totalLines, offset)}, nil
	}
	lines = lines[offset:]

	// Apply limit
	if len(lines) > limit {
		lines = lines[:limit]
	}

	// Format with line numbers and truncate long lines
	var builder strings.Builder
	for i, line := range lines {
		lineNum := offset + i + 1
		if len(line) > maxLineLength {
			line = line[:maxLineLength] + "..."
		}
		fmt.Fprintf(&builder, "%4d | %s\n", lineNum, line)
	}

	summary := builder.String()
	if offset+len(lines) < totalLines {
		summary += fmt.Sprintf("\n(显示 %d-%d 行，共 %d 行。使用 offset=%d 查看更多)", offset+1, offset+len(lines), totalLines, offset+len(lines))
	}

	return Result{
		Summary: summary,
		Data: map[string]any{
			"path":        filePath,
			"total_lines": totalLines,
			"shown_from":  offset + 1,
			"shown_to":    offset + len(lines),
		},
	}, nil
}
