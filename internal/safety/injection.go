package safety

import (
	"regexp"
	"strings"
)

// Severity levels for injection detection.
type Severity int

const (
	SeverityCritical Severity = 4
	SeverityHigh     Severity = 3
	SeverityMedium   Severity = 2
	SeverityLow      Severity = 1
)

// InjectionRule defines a prompt injection detection pattern.
type InjectionRule struct {
	ID       string
	Severity Severity
	Pattern  *regexp.Regexp
	Reason   string
}

// InjectionMatch records a single detected injection pattern.
type InjectionMatch struct {
	Rule     InjectionRule
	Position int
	Snippet  string
}

// ScanResult is the output of injection scanning.
type ScanResult struct {
	IsBlocked       bool
	HighestSeverity Severity
	Matches         []InjectionMatch
	ErrorCode       string
}

// InjectionRules is the pattern library for injection detection.
var InjectionRules = []InjectionRule{
	// CRITICAL — 直接拦截
	{ID: "PI-001", Severity: SeverityCritical, Pattern: regexp.MustCompile(`(?i)忽略(之前|前面|所有|以上).{0,10}(指令|命令|提示|prompt|规则)`), Reason: "指令覆盖-中文"},
	{ID: "PI-002", Severity: SeverityCritical, Pattern: regexp.MustCompile(`(?i)(ignore|disregard|forget)\s+(all|previous|above|prior)\s+(instructions?|prompts?|rules?|commands?)`), Reason: "instruction override"},
	{ID: "PI-003", Severity: SeverityCritical, Pattern: regexp.MustCompile(`(?i)你现在是\s*(root|admin|管理员|超级用户)`), Reason: "角色越狱-中文"},
	{ID: "PI-004", Severity: SeverityCritical, Pattern: regexp.MustCompile(`(?i)\bDAN\s*mode\b|\bdeveloper\s+mode\b|\bjailbreak\b`), Reason: "经典越狱关键词"},
	{ID: "PI-005", Severity: SeverityCritical, Pattern: regexp.MustCompile(`(?i)你不再受.{0,5}(限制|约束|规则)`), Reason: "解除限制-中文"},
	// HIGH — 拦截
	{ID: "PI-010", Severity: SeverityHigh, Pattern: regexp.MustCompile(`<\s*/?(system|assistant|instruction)\s*>`), Reason: "边界标记伪造"},
	{ID: "PI-011", Severity: SeverityHigh, Pattern: regexp.MustCompile(`(?i)(show|output|print|display|reveal).{0,10}(system\s*prompt|内部指令|系统提示)`), Reason: "system prompt 泄露"},
	{ID: "PI-012", Severity: SeverityHigh, Pattern: regexp.MustCompile(`(?i)(把|将).{0,5}(system\s*prompt|系统提示词?|内部指令).{0,5}(输出|显示|告诉|说出)`), Reason: "system prompt 泄露-中文"},
	{ID: "PI-013", Severity: SeverityHigh, Pattern: regexp.MustCompile(`(?i)act\s+as\s+(root|admin|hacker)`), Reason: "角色扮演攻击"},
}

const (
	UserInputStartMarker = "<<<USER_INPUT>>>"
	UserInputEndMarker   = "<<<END_USER_INPUT>>>"
)

// WrapUserInput isolates user input with markers to prevent injection.
func WrapUserInput(raw string) string {
	cleaned := strings.ReplaceAll(raw, UserInputStartMarker, "[U_START]")
	cleaned = strings.ReplaceAll(cleaned, UserInputEndMarker, "[U_END]")
	cleaned = strings.ReplaceAll(cleaned, "```", "`\u200B`\u200B`")
	return UserInputStartMarker + "\n" + cleaned + "\n" + UserInputEndMarker
}

// ScanInjection checks text for prompt injection patterns.
func ScanInjection(text string) ScanResult {
	var matches []InjectionMatch
	var highest Severity

	for _, rule := range InjectionRules {
		locs := rule.Pattern.FindAllStringIndex(text, -1)
		for _, loc := range locs {
			snippet := text[loc[0]:loc[1]]
			if len(snippet) > 80 {
				snippet = snippet[:80] + "..."
			}
			matches = append(matches, InjectionMatch{
				Rule:     rule,
				Position: loc[0],
				Snippet:  snippet,
			})
			if rule.Severity > highest {
				highest = rule.Severity
			}
		}
	}

	isBlocked := highest >= SeverityHigh
	errCode := ""
	if isBlocked {
		errCode = "SAFETY_INJECT_001"
	}

	return ScanResult{
		IsBlocked:       isBlocked,
		HighestSeverity: highest,
		Matches:         matches,
		ErrorCode:       errCode,
	}
}
