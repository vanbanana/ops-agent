package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadToolOutputTool allows the LLM to read back persisted large tool outputs.
// When a tool's output exceeds 30KB, it's saved to disk and only a 2KB preview
// is kept in the message history. This tool lets the LLM read specific portions
// of the full output on demand.
type ReadToolOutputTool struct{}

func NewReadToolOutputTool() *ReadToolOutputTool { return &ReadToolOutputTool{} }

func (t *ReadToolOutputTool) Name() string { return "read_tool_output" }

func (t *ReadToolOutputTool) Description() string {
	return `读取之前被截断保存到磁盘的工具完整输出。

当工具输出超过 30KB 时，完整内容会保存到磁盘文件，消息中只保留前 2KB 预览。
使用此工具可以读取完整输出的指定范围。

参数:
- path: 保存的文件路径（在输出预览中会显示）
- start_line: 起始行号（从1开始，默认1）
- end_line: 结束行号（默认为文件末尾，最多返回500行）`
}

func (t *ReadToolOutputTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "工具输出文件的路径",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "起始行号（从1开始，默认1）",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": "结束行号（默认文件末尾，单次最多500行）",
			},
		},
		"required": []string{"path"},
	}
}

func (t *ReadToolOutputTool) Type() ToolType { return ToolReadOnly }

func (t *ReadToolOutputTool) Execute(_ context.Context, args map[string]any) (Result, error) {
	path, _ := args["path"].(string)
	if path == "" {
		return Result{Error: "path parameter is required"}, nil
	}

	// Security: only allow reading from the tool-outputs directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Result{Error: "invalid path"}, nil
	}
	outputDir, _ := filepath.Abs("data/tool-outputs")
	if !strings.HasPrefix(absPath, outputDir) {
		return Result{Error: "access denied: can only read from data/tool-outputs/"}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Error: fmt.Sprintf("cannot read file: %v", err)}, nil
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	startLine := 1
	if v, ok := args["start_line"].(float64); ok && v > 0 {
		startLine = int(v)
	}
	endLine := totalLines
	if v, ok := args["end_line"].(float64); ok && v > 0 {
		endLine = int(v)
	}

	// Clamp
	if startLine < 1 {
		startLine = 1
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine > endLine {
		return Result{Error: "start_line > end_line"}, nil
	}

	// Max 500 lines per read
	maxLines := 500
	if endLine-startLine+1 > maxLines {
		endLine = startLine + maxLines - 1
	}

	selected := lines[startLine-1 : endLine]
	content := strings.Join(selected, "\n")

	summary := fmt.Sprintf("[行 %d-%d / 共 %d 行]\n%s", startLine, endLine, totalLines, content)
	if endLine < totalLines {
		summary += fmt.Sprintf("\n\n[还有 %d 行未显示，使用 start_line=%d 继续读取]",
			totalLines-endLine, endLine+1)
	}

	return Result{Summary: summary}, nil
}
