package safety

import "strings"

// FlagSeverity indicates how dangerous a flag is.
type FlagSeverity int

const (
	FlagSafe      FlagSeverity = iota // Explicitly safe
	FlagDangerous                     // Blocked outright
	FlagWarning                       // Allowed but with warning
)

// FlagSpec defines the safety semantics of a specific flag for a command.
type FlagSpec struct {
	Flag     string       // The flag string (e.g. "-9", "--force", "-delete")
	Severity FlagSeverity // How dangerous this flag is
	Reason   string       // Human-readable explanation
}

// CommandFlagSpecs maps command names to their flag-level safety rules.
// Only dangerous/warning flags need to be listed; unlisted flags are
// checked against the existing whitelist ArgPrefixWhitelist/ForbiddenArgs.
var CommandFlagSpecs = map[string][]FlagSpec{
	"kill": {
		{Flag: "-9", Severity: FlagDangerous, Reason: "SIGKILL 强制杀死进程，无法清理资源"},
		{Flag: "-KILL", Severity: FlagDangerous, Reason: "SIGKILL 强制杀死进程"},
		{Flag: "-SIGKILL", Severity: FlagDangerous, Reason: "SIGKILL 强制杀死进程"},
		{Flag: "-15", Severity: FlagWarning, Reason: "SIGTERM 终止进程（允许清理）"},
		{Flag: "-TERM", Severity: FlagWarning, Reason: "SIGTERM 终止进程（允许清理）"},
		{Flag: "-HUP", Severity: FlagWarning, Reason: "SIGHUP 重载配置/挂断"},
	},
	"find": {
		{Flag: "-delete", Severity: FlagDangerous, Reason: "递归删除匹配的文件"},
		{Flag: "-exec", Severity: FlagDangerous, Reason: "对匹配文件执行任意命令"},
		{Flag: "-execdir", Severity: FlagDangerous, Reason: "在文件目录下执行任意命令"},
		{Flag: "-ok", Severity: FlagDangerous, Reason: "交互式执行任意命令"},
	},
	"systemctl": {
		{Flag: "mask", Severity: FlagDangerous, Reason: "永久屏蔽服务，无法启动"},
		{Flag: "unmask", Severity: FlagWarning, Reason: "解除服务屏蔽"},
		{Flag: "disable", Severity: FlagWarning, Reason: "禁止服务开机自启"},
		{Flag: "enable", Severity: FlagWarning, Reason: "设置服务开机自启"},
		{Flag: "restart", Severity: FlagWarning, Reason: "将重启正在运行的服务"},
		{Flag: "stop", Severity: FlagWarning, Reason: "将停止正在运行的服务"},
		{Flag: "daemon-reload", Severity: FlagWarning, Reason: "重载 systemd 配置"},
	},
	"journalctl": {
		{Flag: "--vacuum-size", Severity: FlagDangerous, Reason: "删除日志直到总大小低于指定值"},
		{Flag: "--vacuum-time", Severity: FlagDangerous, Reason: "删除早于指定时间的日志"},
		{Flag: "--vacuum-files", Severity: FlagDangerous, Reason: "删除日志文件直到数量低于指定值"},
		{Flag: "--rotate", Severity: FlagWarning, Reason: "强制轮转当前日志"},
		{Flag: "--flush", Severity: FlagWarning, Reason: "刷新日志到持久存储"},
	},
	"chmod": {
		{Flag: "-R", Severity: FlagWarning, Reason: "递归修改权限"},
		{Flag: "--recursive", Severity: FlagWarning, Reason: "递归修改权限"},
	},
	"chown": {
		{Flag: "-R", Severity: FlagWarning, Reason: "递归修改属主"},
		{Flag: "--recursive", Severity: FlagWarning, Reason: "递归修改属主"},
	},
	"rm": {
		{Flag: "-r", Severity: FlagDangerous, Reason: "递归删除目录"},
		{Flag: "-R", Severity: FlagDangerous, Reason: "递归删除目录"},
		{Flag: "--recursive", Severity: FlagDangerous, Reason: "递归删除目录"},
		{Flag: "-f", Severity: FlagWarning, Reason: "强制删除，无确认提示"},
		{Flag: "--force", Severity: FlagWarning, Reason: "强制删除，无确认提示"},
	},
	"dd": {
		{Flag: "of=/dev/", Severity: FlagDangerous, Reason: "直接写入块设备"},
	},
	"iptables": {
		{Flag: "-F", Severity: FlagDangerous, Reason: "清空所有防火墙规则"},
		{Flag: "--flush", Severity: FlagDangerous, Reason: "清空所有防火墙规则"},
		{Flag: "-X", Severity: FlagWarning, Reason: "删除自定义链"},
		{Flag: "-D", Severity: FlagWarning, Reason: "删除规则"},
	},
	"reboot": {
		{Flag: "", Severity: FlagDangerous, Reason: "重启服务器"},
	},
	"shutdown": {
		{Flag: "", Severity: FlagDangerous, Reason: "关闭服务器"},
	},
	"halt": {
		{Flag: "", Severity: FlagDangerous, Reason: "停止服务器"},
	},
	"poweroff": {
		{Flag: "", Severity: FlagDangerous, Reason: "关闭服务器电源"},
	},
}

// CheckFlags validates command arguments against the flag spec table.
// Returns:
//   - blocked=true if any FlagDangerous flag is found
//   - warning (non-empty) if any FlagWarning flag is found
//   - reason explains why it was blocked
func CheckFlags(cmdName string, args []string) (blocked bool, warning string, reason string) {
	specs, hasSpecs := CommandFlagSpecs[cmdName]
	if !hasSpecs {
		return false, "", ""
	}

	var warnings []string

	for _, spec := range specs {
		if spec.Flag == "" {
			// Empty flag means the command itself is dangerous (e.g. reboot)
			if spec.Severity == FlagDangerous {
				return true, "", spec.Reason
			}
			if spec.Severity == FlagWarning {
				warnings = append(warnings, spec.Reason)
			}
			continue
		}

		for _, arg := range args {
			// Exact match or prefix match (for dd of=/dev/sda)
			matched := arg == spec.Flag ||
				strings.HasPrefix(arg, spec.Flag)

			// Also check -s KILL pattern (kill -s KILL)
			if cmdName == "kill" && arg == "-s" {
				// Check next arg — but we only have current arg here.
				// Handle -s combined: covered by checking if next iteration matches signal name
			}

			if matched {
				switch spec.Severity {
				case FlagDangerous:
					return true, "", spec.Reason
				case FlagWarning:
					warnings = append(warnings, spec.Reason)
				}
			}
		}
	}

	// Special case: kill -s KILL / kill -s 9
	if cmdName == "kill" {
		for i, arg := range args {
			if (arg == "-s" || arg == "--signal") && i+1 < len(args) {
				sigArg := strings.ToUpper(args[i+1])
				if sigArg == "KILL" || sigArg == "SIGKILL" || sigArg == "9" {
					return true, "", "kill -s KILL/9: SIGKILL 强制杀死进程"
				}
			}
		}
	}

	if len(warnings) > 0 {
		return false, strings.Join(warnings, "; "), ""
	}
	return false, "", ""
}
