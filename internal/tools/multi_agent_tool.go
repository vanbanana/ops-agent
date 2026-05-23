package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MultiAgentExecutor is the interface the multi-agent orchestrator must implement.
// This decouples tools package from agent package (dependency direction: tools → interface ← agent).
type MultiAgentExecutor interface {
	// RunSync executes multi-agent analysis synchronously and returns the synthesized result.
	RunSync(ctx context.Context, question string) (string, error)
}

// MultiAgentTool exposes multi-agent orchestration as a tool callable by the LLM.
// Design pattern: same as OpenCode's AgentTool — the model decides when to use it.
type MultiAgentTool struct {
	executor MultiAgentExecutor
}

// NewMultiAgentTool creates the multi-agent analysis tool.
func NewMultiAgentTool(executor MultiAgentExecutor) *MultiAgentTool {
	return &MultiAgentTool{executor: executor}
}

func (t *MultiAgentTool) Name() string { return "multi_agent_analyze" }

func (t *MultiAgentTool) Description() string {
	return `启动多Agent协作分析。当用户问题需要从多个系统维度收集信息并综合分析时使用。

适用场景:
- 综合性能排查（CPU + 内存 + 磁盘 + 网络 + 进程）
- 故障诊断（需要交叉比对多个数据源）
- 全面巡检/健康检查
- "为什么系统变慢了"类需要多角度分析的问题

不适用（直接使用对应探针工具）:
- 单一操作：看磁盘、重启服务、查日志、看进程
- 明确目标：某个进程占用多少内存、某个端口被谁占了

此工具会启动 Planner→Executor→Verifier 流程，可能需要 10-30 秒完成。`
}

func (t *MultiAgentTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "需要多维度分析的运维问题，用自然语言描述",
			},
		},
		"required": []string{"question"},
	}
}

func (t *MultiAgentTool) Type() ToolType { return ToolExternal }

func (t *MultiAgentTool) Execute(ctx context.Context, args map[string]any) (Result, error) {
	question, _ := args["question"].(string)
	if question == "" {
		return Result{Error: "question parameter is required"}, nil
	}

	// Apply timeout for multi-agent execution (longer because it does multiple LLM calls internally)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	result, err := t.executor.RunSync(ctx, question)
	if err != nil {
		return Result{Error: fmt.Sprintf("multi-agent analysis failed: %v", err)}, nil
	}

	// Truncate if too long
	if len(result) > 8192 {
		result = result[:8192] + "\n[结果已截断]"
	}

	return Result{
		Summary: result,
		Data:    map[string]any{"question": question, "result_length": len(result)},
	}, nil
}

// MultiAgentArgsFromJSON parses tool call arguments.
func MultiAgentArgsFromJSON(argsJSON string) (string, error) {
	var args struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	if strings.TrimSpace(args.Question) == "" {
		return "", fmt.Errorf("question is empty")
	}
	return args.Question, nil
}
