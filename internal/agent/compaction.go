package agent

import "fmt"

// compactMessages manages context window by summarizing old conversations.
// Strategy (mirrors OpenCode's approach):
// 1. If within budget: do nothing
// 2. Truncate long tool outputs in older messages
// 3. If still over: replace old history with a structured summary, keep recent rounds
func compactMessages(msgs []Message, tokenBudget int) []Message {
	total := estimateTokens(msgs)
	if total <= tokenBudget {
		return msgs
	}

	// Phase 1: Truncate old tool message content (keep last 3 rounds intact)
	protectedStart := len(msgs) - 6 // protect last ~3 rounds (user+assistant+tool pairs)
	if protectedStart < 1 {
		protectedStart = 1
	}

	for i := 1; i < protectedStart; i++ {
		if msgs[i].Role == "tool" && len(msgs[i].Content) > 300 {
			// Keep first 150 chars (usually the key finding) + truncation notice
			msgs[i].Content = msgs[i].Content[:150] + fmt.Sprintf("\n...[已压缩，原%d字符]", len(msgs[i].Content))
		}
	}

	total = estimateTokens(msgs)
	if total <= tokenBudget {
		return msgs
	}

	// Phase 2: Build a structured summary of old conversation
	// Keep: system prompt (msgs[0]) + summary + last N messages
	keepLast := 8 // keep last 8 messages (~4 rounds)
	if keepLast > len(msgs)-1 {
		keepLast = len(msgs) - 1
	}

	// Build summary from dropped messages
	summary := buildConversationSummary(msgs[1 : len(msgs)-keepLast])

	compacted := make([]Message, 0, keepLast+2)
	compacted = append(compacted, msgs[0]) // system prompt
	compacted = append(compacted, Message{
		Role:    "user",
		Content: fmt.Sprintf("[对话摘要 — 之前%d轮已压缩]\n%s", (len(msgs)-keepLast-1)/2, summary),
	})
	compacted = append(compacted, msgs[len(msgs)-keepLast:]...)
	return compacted
}

// buildConversationSummary creates a concise summary of conversation history.
func buildConversationSummary(msgs []Message) string {
	var userQuestions []string
	var toolsUsed []string
	var keyFindings []string
	toolSet := map[string]bool{}

	for _, m := range msgs {
		switch m.Role {
		case "user":
			if len(m.Content) > 0 && len(userQuestions) < 3 {
				q := m.Content
				if len(q) > 60 {
					q = q[:60] + "..."
				}
				userQuestions = append(userQuestions, q)
			}
		case "tool":
			// Extract key info from tool results
			if len(m.Content) > 0 && len(keyFindings) < 5 {
				finding := m.Content
				if len(finding) > 100 {
					finding = finding[:100] + "..."
				}
				keyFindings = append(keyFindings, finding)
			}
		case "assistant":
			for _, tc := range m.ToolCalls {
				if !toolSet[tc.Function.Name] {
					toolSet[tc.Function.Name] = true
					toolsUsed = append(toolsUsed, tc.Function.Name)
				}
			}
		}
	}

	result := ""
	if len(userQuestions) > 0 {
		result += "用户问题: " + joinStrings(userQuestions, "; ") + "\n"
	}
	if len(toolsUsed) > 0 {
		result += "已使用工具: " + joinStrings(toolsUsed, ", ") + "\n"
	}
	if len(keyFindings) > 0 {
		result += "关键发现:\n"
		for _, f := range keyFindings {
			result += "- " + f + "\n"
		}
	}
	return result
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// estimateTokens roughly estimates token count (4 bytes per token for mixed content).
func estimateTokens(msgs []Message) int {
	total := 0
	for _, m := range msgs {
		total += len(m.Content) / 4
		for _, tc := range m.ToolCalls {
			total += len(tc.Function.Arguments) / 4
		}
	}
	return total
}
