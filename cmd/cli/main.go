package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"ops-agent/internal/agent"
	"ops-agent/internal/config"
	"ops-agent/internal/safety"
	"ops-agent/internal/store"
	"ops-agent/internal/tools"
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

func main() {
	config.LoadDotEnv(".env")

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s配置错误: %v%s\n", colorRed, err, colorReset)
		fmt.Fprintf(os.Stderr, "请确保 .env 文件存在且配置了 LLM_API_KEY 和 LLM_BASE_URL\n")
		os.Exit(1)
	}

	llmClient := agent.NewLLMClient(agent.ClientConfig{
		BaseURL: cfg.LLMBaseURL,
		APIKey:  cfg.LLMAPIKey,
		Model:   cfg.LLMModel,
		Timeout: 60 * time.Second,
	})

	registry := tools.NewRegistry()
	tools.RegisterAllProbes(registry)

	agentInstance := agent.NewAgent(llmClient, registry, agent.AgentConfig{
		Model: cfg.LLMModel,
	}, nil) // CLI mode: no permission service (auto-approve)

	sessionID := fmt.Sprintf("sess_%d", time.Now().UnixNano())

	sessions := store.NewSessionStore()
	sessions.GetOrCreate(sessionID)

	fmt.Printf("%s%s╔══════════════════════════════════════════╗%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s║   🖥️  Linux 运维智能体 — CLI 对话模式    ║%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════╝%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s  LLM: %s | 工具: %d 个探针%s\n", colorDim, cfg.LLMModel, len(registry.List()), colorReset)
	fmt.Printf("%s  输入 /quit 退出 | /tools 查看工具 | /safety <cmd> 测试安全%s\n\n", colorDim, colorReset)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("%s%s你> %s", colorBold, colorGreen, colorReset)
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Commands
		if input == "/quit" || input == "/exit" {
			fmt.Printf("%s再见！%s\n", colorCyan, colorReset)
			break
		}
		if input == "/tools" {
			fmt.Printf("\n%s已注册工具:%s\n", colorYellow, colorReset)
			for i, t := range registry.List() {
				fmt.Printf("  %d. %s%s%s — %s\n", i+1, colorCyan, t.Name(), colorReset, t.Description())
			}
			fmt.Println()
			continue
		}
		if strings.HasPrefix(input, "/safety ") {
			cmd := strings.TrimPrefix(input, "/safety ")
			testSafety(cmd)
			continue
		}

		// Run agent
		runChat(agentInstance, sessions, sessionID, input)
	}
}

func runChat(agentInstance *agent.Agent, sessions *store.SessionStore, sessionID, message string) {
	ctx := context.Background()

	// Load history and store user message
	history := storeToAgentMessages(sessions.GetRecentMessages(sessionID, 20))
	sessions.AppendMessage(sessionID, store.Message{Role: "user", Content: message})

	events := agentInstance.RunStream(ctx, sessionID, message, history)

	fmt.Println()
	for event := range events {
		switch event.Type {
		case "mode_decision":
			mode, _ := event.Data["mode"].(string)
			reason, _ := event.Data["reason"].(string)
			if mode == "multi" {
				fmt.Printf("  %s🔀 多Agent模式: %s%s\n", colorYellow, reason, colorReset)
			}
		case "agent_role":
			role, _ := event.Data["role"].(string)
			subTask, _ := event.Data["sub_task"].(string)
			msg, _ := event.Data["message"].(string)
			switch role {
			case "planner":
				fmt.Printf("\n  %s📋 [Planner] 正在拆解子任务...%s\n", colorCyan, colorReset)
			case "coordinator":
				fmt.Printf("  %s📡 [Coordinator] %s%s\n", colorCyan, msg, colorReset)
			case "executor":
				execID, _ := event.Data["executor_id"].(float64)
				fmt.Printf("  %s⚡ [Executor #%.0f] %s%s\n", colorBlue, execID, subTask, colorReset)
			case "verifier":
				fmt.Printf("  %s🔍 [Verifier] 验证分析结果...%s\n", colorYellow, colorReset)
			}
		case "verifier_result":
			verified, _ := event.Data["verified"].(bool)
			reason, _ := event.Data["reason"].(string)
			if verified {
				fmt.Printf("  %s✅ 验证通过: %s%s\n", colorGreen, reason, colorReset)
			} else {
				fmt.Printf("  %s⚠️  验证未通过: %s%s\n", colorYellow, reason, colorReset)
			}
		case "sense":
			status, _ := event.Data["status"].(string)
			if status == "blocked" {
				reason, _ := event.Data["reason"].(string)
				fmt.Printf("  %s🚫 注入拦截: %s%s\n\n", colorRed, reason, colorReset)
				return
			}
		case "plan":
			if toolsData, ok := event.Data["tools"].([]map[string]any); ok {
				for _, t := range toolsData {
					fmt.Printf("  %s🔧 调用工具: %s%s\n", colorYellow, t["name"], colorReset)
				}
			}
		case "execute_done":
			tool, _ := event.Data["tool"].(string)
			status, _ := event.Data["status"].(string)
			icon := "✅"
			if status != "ok" {
				icon = "❌"
			}
			fmt.Printf("  %s%s %s 完成%s\n", colorDim, icon, tool, colorReset)
		case "output":
			reply, _ := event.Data["reply"].(string)
			fmt.Printf("\n%s%s助手>%s %s\n\n", colorBold, colorBlue, colorReset, reply)
			// Store assistant reply for context
			sessions.AppendMessage(sessionID, store.Message{Role: "assistant", Content: reply})
		case "error":
			errCode, _ := event.Data["error_code"].(string)
			msg, _ := event.Data["message"].(string)
			fmt.Printf("  %s❌ 错误 [%s]: %s%s\n\n", colorRed, errCode, msg, colorReset)
		}
	}
}

func testSafety(cmd string) {
	fmt.Printf("\n  %s测试命令:%s %s\n", colorDim, colorReset, cmd)
	result := safety.ValidateCommand(cmd)
	switch result.Status {
	case safety.StatusPassed:
		fmt.Printf("  %s✅ PASSED%s\n\n", colorGreen, colorReset)
	case safety.StatusBlocked:
		fmt.Printf("  %s🚫 BLOCKED [%s]: %s%s\n\n", colorRed, result.Reason, result.Detail, colorReset)
	case safety.StatusEscalate:
		fmt.Printf("  %s⚠️  ESCALATE: %s%s\n\n", colorYellow, result.Detail, colorReset)
	}
}

func safetyCheck(cmd string) string {
	// Direct import would be cleaner but let's keep it simple
	// by calling the internal package
	return fmt.Sprintf("(请用 curl 'http://localhost:8080/api/v1/safety/scan?cmd=%s' 测试)", cmd)
}


func storeToAgentMessages(msgs []store.Message) []agent.Message {
	result := make([]agent.Message, 0, len(msgs))
	for _, m := range msgs {
		result = append(result, agent.Message{Role: m.Role, Content: m.Content})
	}
	return result
}
