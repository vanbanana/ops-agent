// Package prompt manages system prompts for the ops-agent.
// Architecture mirrors OpenCode: base prompt hardcoded in code + project-specific
// context loaded from .ops-agent/ directory at runtime.
package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// AgentRole identifies which agent prompt to use.
type AgentRole string

const (
	RoleCoder      AgentRole = "coder"
	RolePlanner    AgentRole = "planner"
	RoleExecutor   AgentRole = "executor"
	RoleVerifier   AgentRole = "verifier"
	RoleSummarizer AgentRole = "summarizer"
)

// GetPrompt returns the full system prompt for the given agent role.
// It combines: base prompt (hardcoded) + environment info (dynamic) + project context (.ops-agent/).
func GetPrompt(role AgentRole, workDir string) string {
	base := getBasePrompt(role)
	env := getEnvironmentInfo(workDir)
	projectCtx := getProjectContext(workDir, role)

	parts := []string{base}
	if env != "" {
		parts = append(parts, env)
	}
	if projectCtx != "" {
		parts = append(parts, "# 项目自定义指令\n"+projectCtx)
	}
	return strings.Join(parts, "\n\n")
}

func getBasePrompt(role AgentRole) string {
	switch role {
	case RoleCoder:
		return baseCoderPrompt
	case RolePlanner:
		return basePlannerPrompt
	case RoleExecutor:
		return baseExecutorPrompt
	case RoleVerifier:
		return baseVerifierPrompt
	case RoleSummarizer:
		return baseSummarizerPrompt
	default:
		return baseCoderPrompt
	}
}

func getEnvironmentInfo(workDir string) string {
	platform := runtime.GOOS
	arch := runtime.GOARCH
	date := time.Now().Format("2006-01-02 15:04")
	hostname, _ := os.Hostname()

	// Detect if target is likely a server or local dev machine
	serverHint := ""
	if _, err := os.Stat("/etc/systemd/system"); err == nil {
		serverHint = "systemd: 可用"
	}
	if _, err := os.Stat("/var/log/journal"); err == nil {
		if serverHint != "" {
			serverHint += ", "
		}
		serverHint += "journald: 可用"
	}

	return fmt.Sprintf(`<env>
主机: %s
平台: %s/%s
时间: %s
系统服务: %s
</env>`, hostname, platform, arch, date, serverHint)
}

// getProjectContext reads prompt customization files from .ops-agent/ directory.
// Mirrors OpenCode's contextPaths mechanism.
func getProjectContext(workDir string, role AgentRole) string {
	var parts []string

	// 1. Read the main project instructions file (.ops-agent/AGENT.md)
	// This is equivalent to OpenCode's "OpenCode.md" / "CLAUDE.md"
	mainInstructions := readFileIfExists(filepath.Join(workDir, ".ops-agent", "AGENT.md"))
	if mainInstructions != "" {
		parts = append(parts, mainInstructions)
	}

	// 2. Read role-specific prompt override (.ops-agent/agents/<role>.md)
	roleFile := readFileIfExists(filepath.Join(workDir, ".ops-agent", "agents", string(role)+".md"))
	if roleFile != "" {
		parts = append(parts, fmt.Sprintf("# %s 角色指令\n%s", role, roleFile))
	}

	// 3. Read all files in .ops-agent/prompts/ directory (shared context)
	promptsDir := filepath.Join(workDir, ".ops-agent", "prompts")
	entries, err := os.ReadDir(promptsDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content := readFileIfExists(filepath.Join(promptsDir, entry.Name()))
			if content != "" {
				parts = append(parts, fmt.Sprintf("# From: .ops-agent/prompts/%s\n%s", entry.Name(), content))
			}
		}
	}

	return strings.Join(parts, "\n\n")
}

func readFileIfExists(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return ""
	}
	return content
}
