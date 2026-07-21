package main

import (
	"encoding/json"
	"fmt"
	"os"

	"ops-agent/internal/agent"
)

// renderer formats agent events for the terminal.
type renderer struct {
	format string
}

func newRenderer(format string) *renderer {
	if format != "json" {
		format = "text"
	}
	return &renderer{format: format}
}

// Event prints a single agent event.
func (r *renderer) Event(event agent.Event) {
	if r.format == "json" {
		_ = json.NewEncoder(os.Stdout).Encode(event)
		return
	}

	switch event.Type {
	case "start":
		// Suppress verbose start events in text mode.
	case "sense":
		status, _ := event.Data["status"].(string)
		if status == "blocked" {
			reason, _ := event.Data["reason"].(string)
			fmt.Printf("\n  %s🚫 注入拦截: %s%s\n", colorRed, reason, colorReset)
		}
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
	case "text_delta":
		delta, _ := event.Data["delta"].(string)
		fmt.Print(delta)
	case "reasoning_delta":
		delta, _ := event.Data["delta"].(string)
		fmt.Printf("%s%s%s", colorDim, delta, colorReset)
	case "output":
		reply, _ := event.Data["reply"].(string)
		fmt.Printf("\n%s%s助手>%s %s\n\n", colorBold, colorBlue, colorReset, reply)
	case "error":
		errCode, _ := event.Data["error_code"].(string)
		msg, _ := event.Data["message"].(string)
		fmt.Printf("  %s❌ 错误 [%s]: %s%s\n\n", colorRed, errCode, msg, colorReset)
	case "done":
		// Final event; text reply already printed by output.
	}
}
