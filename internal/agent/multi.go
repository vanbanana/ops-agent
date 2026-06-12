package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"ops-agent/internal/agent/prompt"
	"ops-agent/internal/tools"
)

const (
	MaxMultiIterations = 3
	MaxSubtaskRounds   = 8
	MaxMultiTokens     = 64000
)

// Subtask is a unit of work from the Planner.
type Subtask struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Tools       string `json:"tools"`
}

// VerifierResult is the Verifier's assessment.
type VerifierResult struct {
	Verified    bool     `json:"verified"`
	Reason      string   `json:"reason"`
	Confidence  float64  `json:"confidence"`
	MissingInfo []string `json:"missing_info"`
}

// MultiAgent orchestrates Planner/Executor/Verifier.
type MultiAgent struct {
	llm      LLMClient
	registry tools.ToolRegistry
	cfg      AgentConfig
}

// NewMultiAgent creates a multi-agent orchestrator.
func NewMultiAgent(llm LLMClient, reg tools.ToolRegistry, cfg AgentConfig) *MultiAgent {
	return &MultiAgent{llm: llm, registry: reg, cfg: cfg}
}

// RunSync executes multi-agent orchestration synchronously and returns the synthesized result.
// This is the interface used by MultiAgentTool — the LLM calls this via tool_call.
// Implements tools.MultiAgentExecutor interface.
func (m *MultiAgent) RunSync(ctx context.Context, question string) (string, error) {
	var allFindings []string

	for iter := 1; iter <= MaxMultiIterations; iter++ {
		subtasks, _, err := m.plan(ctx, question, allFindings)
		if err != nil {
			return "", fmt.Errorf("planner failed: %w", err)
		}

		// Execute subtasks sequentially (no SSE channel needed in sync mode)
		for _, st := range subtasks {
			finding, execErr := m.executeSubtaskSync(ctx, st)
			if execErr != nil {
				allFindings = append(allFindings, fmt.Sprintf("[%s] 执行失败: %s", st.Description, execErr.Error()))
			} else {
				allFindings = append(allFindings, fmt.Sprintf("[%s] %s", st.Description, finding))
			}
		}

		// Verify
		vr, _, err := m.verify(ctx, question, allFindings)
		if err != nil {
			vr = &VerifierResult{Verified: false, Reason: "Verifier error: " + err.Error()}
		}

		if vr.Verified {
			return m.synthesize(question, allFindings), nil
		}
		// Not verified: next iteration will fill gaps
	}

	// Max iterations: return best-effort
	result := m.synthesize(question, allFindings)
	return result + "\n\n（注：多轮验证未完全通过，以上为尽力而为的分析结果）", nil
}

// executeSubtaskSync runs a subtask without SSE events.
func (m *MultiAgent) executeSubtaskSync(ctx context.Context, st Subtask) (string, error) {
	workDir, _ := os.Getwd()
	systemPrompt := prompt.GetPrompt(prompt.RoleExecutor, workDir)

	messages := []Message{
		{Role: "system", Content: systemPrompt + "\n\n当前子任务: " + st.Description},
		{Role: "user", Content: st.Description},
	}

	toolDefs := m.buildToolDefs()

	for round := 1; round <= MaxSubtaskRounds; round++ {
		resp, err := m.llm.Chat(ctx, ChatRequest{
			Messages: messages,
			Tools:    toolDefs,
		})
		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("empty response")
		}

		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, nil
		}

		messages = append(messages, choice.Message)

		for _, tc := range choice.Message.ToolCalls {
			args := parseToolArgs(tc.Function.Arguments)
			result := m.registry.Dispatch(ctx, tc.Function.Name, args)

			toolContent := result.Summary
			if result.Error != "" {
				toolContent = "Error: " + result.Error
			}
			if len(toolContent) > 4096 {
				toolContent = toolContent[:4096] + "\n[Truncated]"
			}
			messages = append(messages, Message{Role: "tool", Content: toolContent, ToolCallID: tc.ID})
		}
	}

	return "子任务达到最大轮次", nil
}

// Run executes the multi-agent loop.
func (m *MultiAgent) Run(ctx context.Context, sessionID, traceID, userMessage string, out chan<- Event) {
	var allFindings []string
	var totalTokens int // Task 11.5: token budget tracking

	for iter := 1; iter <= MaxMultiIterations; iter++ {
		// Task 11.5: check token budget before each iteration
		if totalTokens >= MaxMultiTokens {
			out <- Event{Type: "error", Data: map[string]any{
				"error_code": "TOKEN_BUDGET_001",
				"message":    fmt.Sprintf("多Agent总token已达%d(上限%d)，强制收口", totalTokens, MaxMultiTokens),
			}}
			break
		}

		// Planner phase
		out <- Event{Type: "agent_role", Data: map[string]any{"role": "planner", "iteration": iter}}

		subtasks, planTokens, err := m.plan(ctx, userMessage, allFindings)
		if err != nil {
			out <- Event{Type: "error", Data: map[string]any{"error_code": "LLM_SERVICE_001", "message": err.Error()}}
			out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "error"}}
			return
		}

		out <- Event{Type: "plan", Data: map[string]any{
			"round": iter, "tools": formatSubtasks(subtasks),
			"tokens": map[string]any{"prompt": planTokens.PromptTokens, "completion": planTokens.CompletionTokens},
		}}
		totalTokens += planTokens.PromptTokens + planTokens.CompletionTokens

		// Coordinator announces dispatch
		out <- Event{Type: "agent_role", Data: map[string]any{
			"role": "coordinator", "action": "dispatch",
			"message": fmt.Sprintf("收到 Planner 的计划，分配 %d 个子任务给 Executor 并行执行", len(subtasks)),
		}}

		// Executor phase: run subtasks in PARALLEL
		type subtaskResult struct {
			index   int
			finding string
			err     error
		}
		resultCh := make(chan subtaskResult, len(subtasks))
		totalTasks := len(subtasks)

		for i, st := range subtasks {
			go func(idx int, task Subtask) {
				out <- Event{Type: "agent_role", Data: map[string]any{
					"role": "executor", "sub_task": task.Description,
					"executor_id": idx + 1, "total": totalTasks, "parallel": true,
				}}

				finding, err := m.executeSubtask(ctx, task, out)
				resultCh <- subtaskResult{index: idx, finding: finding, err: err}
			}(i, st)
		}

		// Coordinator collects results, announcing progress
		collected := 0
		for range subtasks {
			res := <-resultCh
			collected++
			st := subtasks[res.index]

			if res.err != nil {
				allFindings = append(allFindings, fmt.Sprintf("[%s] 执行失败: %s", st.Description, res.err.Error()))
			} else {
				allFindings = append(allFindings, fmt.Sprintf("[%s] %s", st.Description, res.finding))
			}

			// Coordinator reports collection progress
			if collected < totalTasks {
				pending := totalTasks - collected
				out <- Event{Type: "agent_role", Data: map[string]any{
					"role":    "coordinator",
					"action":  "waiting",
					"message": fmt.Sprintf("已收到 %d/%d 份报告，还在等 %d 个 Executor 的结果...", collected, totalTasks, pending),
				}}
			} else {
				out <- Event{Type: "agent_role", Data: map[string]any{
					"role":    "coordinator",
					"action":  "collected",
					"message": fmt.Sprintf("全部 %d 份报告已收齐，正在整理后转交 Verifier", totalTasks),
				}}
			}
		}

		// Verifier phase
		out <- Event{Type: "agent_role", Data: map[string]any{"role": "verifier"}}

		vr, verifyUsage, err := m.verify(ctx, userMessage, allFindings)
		if err != nil {
			// Verifier error → degrade gracefully
			vr = &VerifierResult{Verified: false, Reason: "Verifier error: " + err.Error()}
		}
		totalTokens += verifyUsage.PromptTokens + verifyUsage.CompletionTokens

		out <- Event{Type: "verifier_result", Data: map[string]any{
			"verified": vr.Verified, "reason": vr.Reason,
			"confidence": vr.Confidence, "missing_info": vr.MissingInfo,
			"iteration": iter,
			"tokens": map[string]any{"prompt": verifyUsage.PromptTokens, "completion": verifyUsage.CompletionTokens},
		}}

		if vr.Verified {
			reply := m.synthesize(userMessage, allFindings)
			out <- Event{Type: "output", Data: map[string]any{"reply": reply, "mode": "multi", "verified": true}}
			out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "ok"}}
			return
		}
		// Not verified: missing_info feeds into next iteration
	}

	// Max iterations reached: best-effort reply
	reply := m.synthesize(userMessage, allFindings)
	out <- Event{Type: "output", Data: map[string]any{
		"reply": reply + "\n\n（注：多轮验证未完全通过，以上为尽力而为的分析结果）",
		"mode": "multi", "verified": false,
	}}
	out <- Event{Type: "done", Data: map[string]any{"trace_id": traceID, "session_id": sessionID, "status": "ok"}}
}

func (m *MultiAgent) plan(ctx context.Context, question string, previousFindings []string) ([]Subtask, Usage, error) {
	workDir, _ := os.Getwd()
	systemPrompt := prompt.GetPrompt(prompt.RolePlanner, workDir)

	userPrompt := fmt.Sprintf(`用户问题: "%s"

请把这个问题拆解为 2-5 个子任务，每个子任务是一次独立的系统探查。
返回 JSON 数组格式: [{"id":"1","description":"检查磁盘使用率","tools":"probe_disk"}]
只返回 JSON，不要其他文字。`, question)

	if len(previousFindings) > 0 {
		userPrompt += "\n\n之前已获取的信息:\n" + strings.Join(previousFindings, "\n")
		userPrompt += "\n\n请根据缺失的信息补充子任务。"
	}

	resp, err := m.llm.Chat(ctx, ChatRequest{
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return nil, Usage{}, err
	}

	if len(resp.Choices) == 0 {
		return nil, Usage{}, fmt.Errorf("planner returned empty response")
	}

	content := resp.Choices[0].Message.Content
	// Extract JSON from response (handle markdown code blocks)
	content = extractJSON(content)

	var subtasks []Subtask
	if err := json.Unmarshal([]byte(content), &subtasks); err != nil {
		// Fallback: single generic subtask
		return []Subtask{{ID: "1", Description: "综合系统检查", Tools: "probe_disk,probe_process,probe_memory"}}, resp.Usage, nil
	}

	// Limit subtasks
	if len(subtasks) > 5 {
		subtasks = subtasks[:5]
	}

	return subtasks, resp.Usage, nil
}

func (m *MultiAgent) executeSubtask(ctx context.Context, st Subtask, out chan<- Event) (string, error) {
	// Use single agent loop with limited rounds for subtask
	workDir, _ := os.Getwd()
	systemPrompt := prompt.GetPrompt(prompt.RoleExecutor, workDir)

	messages := []Message{
		{Role: "system", Content: systemPrompt + "\n\n当前子任务: " + st.Description},
		{Role: "user", Content: st.Description},
	}

	toolDefs := m.buildToolDefs()

	for round := 1; round <= MaxSubtaskRounds; round++ {
		resp, err := m.llm.Chat(ctx, ChatRequest{
			Messages: messages,
			Tools:    toolDefs,
		})
		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("empty response")
		}

		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) == 0 {
			return choice.Message.Content, nil
		}

		messages = append(messages, choice.Message)

		for _, tc := range choice.Message.ToolCalls {
			args := parseToolArgs(tc.Function.Arguments)
			out <- Event{Type: "execute", Data: map[string]any{"tool": tc.Function.Name, "args": args}}

			result := m.registry.Dispatch(ctx, tc.Function.Name, args)
			out <- Event{Type: "execute_done", Data: map[string]any{
				"tool": tc.Function.Name, "status": statusFromResult(result),
				"result_preview": truncateStr(result.Summary, 200),
			}}

			toolContent := result.Summary
			if result.Error != "" {
				toolContent = "Error: " + result.Error
			}
			if len(toolContent) > 4096 {
				toolContent = toolContent[:4096] + "\n[Truncated]"
			}
			messages = append(messages, Message{Role: "tool", Content: toolContent, ToolCallID: tc.ID})
		}
	}

	return "子任务达到最大轮次", nil
}

func (m *MultiAgent) verify(ctx context.Context, question string, findings []string) (*VerifierResult, Usage, error) {
	workDir, _ := os.Getwd()
	systemPrompt := prompt.GetPrompt(prompt.RoleVerifier, workDir)

	userPrompt := fmt.Sprintf(`用户原始问题: "%s"

已收集的分析结果:
%s

请判断信息是否足够回答用户问题。
返回 JSON: {"verified":true/false,"reason":"判断依据","confidence":0.0-1.0,"missing_info":["缺失信息"]}
只返回 JSON。`, question, strings.Join(findings, "\n"))

	resp, err := m.llm.Chat(ctx, ChatRequest{
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return nil, Usage{}, err
	}

	if len(resp.Choices) == 0 {
		return &VerifierResult{Verified: false, Reason: "empty response"}, resp.Usage, nil
	}

	content := extractJSON(resp.Choices[0].Message.Content)
	var vr VerifierResult
	if err := json.Unmarshal([]byte(content), &vr); err != nil {
		// Parse failed → not verified but don't panic
		return &VerifierResult{Verified: false, Reason: "JSON parse error: " + err.Error()}, resp.Usage, nil
	}

	return &vr, resp.Usage, nil
}

func (m *MultiAgent) synthesize(question string, findings []string) string {
	// Simple concatenation for now; in production, call LLM to synthesize
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prompt := fmt.Sprintf(`基于以下分析结果，给用户一个综合回答。用户问题: "%s"

分析结果:
%s

要求: 中文回复，结构清晰，先结论后细节。`, question, strings.Join(findings, "\n"))

	resp, err := m.llm.Chat(ctx, ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是运维分析综合输出者，整合多方面分析给出最终结论。"},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil || len(resp.Choices) == 0 {
		return "综合分析结果:\n" + strings.Join(findings, "\n")
	}

	return resp.Choices[0].Message.Content
}

func (m *MultiAgent) buildToolDefs() []ToolDef {
	defs := m.registry.Definitions()
	result := make([]ToolDef, len(defs))
	for i, d := range defs {
		result[i] = ToolDef{
			Type: d.Type,
			Function: FunctionDef{
				Name:        d.Function.Name,
				Description: d.Function.Description,
				Parameters:  d.Function.Parameters,
			},
		}
	}
	return result
}

func extractJSON(s string) string {
	// Strip markdown code blocks
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}

func formatSubtasks(sts []Subtask) []map[string]any {
	result := make([]map[string]any, len(sts))
	for i, st := range sts {
		result[i] = map[string]any{"name": st.Description, "args": st.Tools}
	}
	return result
}
