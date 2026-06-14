package main

import (
	"bufio"
	"fmt"
	"os"

	"ops-agent/internal/agent"
	"ops-agent/internal/store"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

var (
	version     = "dev"
	bufioReader = bufio.NewReader(os.Stdin)
)

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("ops-agent", version)
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printGlobalHelp()
		return
	}

	app, err := newApp()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s%s%s\n", colorRed, err, colorReset)
		fmt.Fprintf(os.Stderr, "请确保 .env 文件存在且配置了 LLM_API_KEY 和 LLM_BASE_URL\n")
		os.Exit(1)
	}
	defer app.close()

	args := os.Args[1:]
	if len(args) == 0 {
		cmdChat(app, []string{})
		return
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "chat":
		cmdChat(app, rest)
	case "run":
		cmdRun(app, rest)
	case "session":
		cmdSession(app, rest)
	case "tools":
		cmdTools(app, rest)
	case "providers":
		cmdProviders(app, rest)
	case "safety":
		cmdSafety(app, rest)
	case "help":
		printGlobalHelp()
	default:
		// Treat unknown first argument as a message for one-shot run,
		// matching the `opencode <message>` shortcut.
		cmdRun(app, os.Args[1:])
	}
}

func printGlobalHelp() {
	fmt.Printf(`%sops-agent — OpenCode-aligned CLI for the Linux Ops Agent%s

USAGE:
  ops-agent [command] [options]

COMMANDS:
  chat                          Start interactive chat (default)
  run <message>...              Run a one-shot prompt
  session {list|delete|export}  Manage persistent sessions
  tools list                    List registered tools
  providers list                List configured LLM providers
  safety scan <command>         Test command safety rules
  help                          Show this help

GLOBAL OPTIONS:
  -h, --help                    Show help
  -v, --version                 Show version

EXAMPLES:
  ops-agent run "查看磁盘使用情况"
  ops-agent run -m deepseek-v4-flash "查看内存"
  ops-agent chat --auto-approve
  ops-agent session list
`, colorBold, colorReset)
}

// storeToAgentMessages converts store messages to agent messages.
func storeToAgentMessages(msgs []store.Message) []agent.Message {
	result := make([]agent.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, agent.Message{Role: m.Role, Content: m.Content})
	}
	return result
}
