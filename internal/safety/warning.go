package safety

import "regexp"

// DestructiveWarning is a non-blocking informational warning for operations
// that are allowed but carry risk. Shown in permission confirmation UI.
type DestructiveWarning struct {
	Pattern *regexp.Regexp
	Message string
}

// DestructiveWarnings lists patterns that produce informational warnings.
// These do NOT block execution — they only add context to the confirmation dialog.
var DestructiveWarnings = []DestructiveWarning{
	// Service operations
	{Pattern: regexp.MustCompile(`systemctl\s+restart\s+`), Message: "将重启正在运行的服务，可能导致短暂中断"},
	{Pattern: regexp.MustCompile(`systemctl\s+stop\s+`), Message: "将停止服务，依赖此服务的功能将不可用"},
	{Pattern: regexp.MustCompile(`systemctl\s+reload\s+`), Message: "将重载服务配置"},

	// Process termination
	{Pattern: regexp.MustCompile(`kill\s+`), Message: "将终止进程"},

	// File operations
	{Pattern: regexp.MustCompile(`truncate\s+`), Message: "将清空文件内容（不可恢复）"},
	{Pattern: regexp.MustCompile(`rm\s+`), Message: "将删除文件（不可恢复）"},
	{Pattern: regexp.MustCompile(`logrotate\s+`), Message: "将强制轮转日志"},

	// Journal
	{Pattern: regexp.MustCompile(`journalctl\s+--rotate`), Message: "将轮转当前日志文件"},
	{Pattern: regexp.MustCompile(`journalctl\s+--vacuum`), Message: "将删除旧日志文件释放空间"},
}

// GetDestructiveWarning checks if a command matches any destructive warning pattern.
// Returns the warning message if matched, empty string otherwise.
// This is purely informational — it does not affect whether the command is allowed.
func GetDestructiveWarning(cmd string) string {
	for _, dw := range DestructiveWarnings {
		if dw.Pattern.MatchString(cmd) {
			return dw.Message
		}
	}

	// Also check flag-level warnings from the flag spec table
	commands := ParseCommand(cmd)
	for _, sc := range commands {
		_, warning, _ := CheckFlags(sc.Name, sc.Args)
		if warning != "" {
			return warning
		}
	}

	return ""
}
