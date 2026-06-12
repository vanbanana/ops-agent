package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// OutputPersistThreshold is the character count above which tool output
	// is persisted to disk and replaced with a preview in the message history.
	// Mirrors Claude Code's approach: full content on disk, preview in context.
	OutputPersistThreshold = 30000

	// OutputPreviewSize is the number of characters kept inline as preview.
	OutputPreviewSize = 2000

	// OutputStorageDir is the base directory for persisted tool outputs.
	OutputStorageDir = "data/tool-outputs"
)

// PersistOutput writes a large tool output to disk and returns the file path
// and a preview string suitable for inclusion in the message history.
//
// Storage path: data/tool-outputs/{sessionID}/{traceID}_{toolName}.txt
//
// The preview contains the first OutputPreviewSize characters plus a reference
// to the full file. The LLM can use read_tool_output to access the full content.
func PersistOutput(sessionID, traceID, toolName, content string) (path string, preview string, err error) {
	// Sanitize inputs for filesystem safety
	sessionID = sanitizePathComponent(sessionID)
	traceID = sanitizePathComponent(traceID)
	toolName = sanitizePathComponent(toolName)

	dir := filepath.Join(OutputStorageDir, sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("create output dir: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.txt", traceID, toolName)
	path = filepath.Join(dir, filename)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", "", fmt.Errorf("write output file: %w", err)
	}

	// Build preview: first N chars, cut at newline boundary if possible
	previewText := content
	hasMore := false
	if len(content) > OutputPreviewSize {
		previewText = content[:OutputPreviewSize]
		// Try to cut at last newline for cleaner preview
		if lastNL := strings.LastIndex(previewText, "\n"); lastNL > OutputPreviewSize/2 {
			previewText = content[:lastNL]
		}
		hasMore = true
	}

	var sb strings.Builder
	sb.WriteString(previewText)
	if hasMore {
		sb.WriteString(fmt.Sprintf("\n\n...[完整输出已保存: %s, 共 %d 字符]\n", path, len(content)))
		sb.WriteString(fmt.Sprintf("[使用 read_tool_output 工具可读取完整内容]"))
	}

	return path, sb.String(), nil
}

// sanitizePathComponent removes path separators and null bytes from a string.
func sanitizePathComponent(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "\x00", "")
	if s == "" {
		s = "unknown"
	}
	return s
}
