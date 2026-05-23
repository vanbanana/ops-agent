package safety

import "regexp"

// CommandSpec defines per-command security constraints.
type CommandSpec struct {
	ReadOnly           bool
	RequiresPathCheck  bool
	ArgPrefixWhitelist []string // First arg must be in this list (e.g. systemctl status)
	ForbiddenArgs      []string // Any arg matching these → BLOCKED
}

// CommandWhitelist is the exact-match whitelist for argv[0].
var CommandWhitelist = map[string]CommandSpec{
	"df":         {ReadOnly: true},
	"du":         {ReadOnly: true},
	"ls":         {ReadOnly: true},
	"ps":         {ReadOnly: true},
	"ss":         {ReadOnly: true},
	"ip":         {ReadOnly: true, ArgPrefixWhitelist: []string{"addr", "route", "link", "a", "r", "l"}},
	"free":       {ReadOnly: true},
	"uptime":     {ReadOnly: true},
	"uname":      {ReadOnly: true},
	"top":        {ReadOnly: true},
	"cat":        {ReadOnly: true, RequiresPathCheck: true},
	"head":       {ReadOnly: true, RequiresPathCheck: true},
	"tail":       {ReadOnly: true, RequiresPathCheck: true},
	"grep":       {ReadOnly: true, RequiresPathCheck: true},
	"find":       {ReadOnly: true, ForbiddenArgs: []string{"-delete", "-exec"}},
	"lsof":       {ReadOnly: true},
	"wc":         {ReadOnly: true},
	"sort":       {ReadOnly: true},
	"journalctl": {ReadOnly: true, ForbiddenArgs: []string{"--rotate", "--vacuum-files", "--vacuum-time"}},
	"systemctl":  {ReadOnly: false, ArgPrefixWhitelist: []string{"status", "is-active", "list-units", "restart", "start", "stop", "reload"}},
	"truncate":   {ReadOnly: false, RequiresPathCheck: true},
	"kill":       {ReadOnly: false, ForbiddenArgs: []string{"-9", "-KILL", "-SIGKILL"}},
	"logrotate":  {ReadOnly: false},
}

// ProtectedPathPrefixes — write operations to these paths are blocked.
var ProtectedPathPrefixes = []string{
	"/etc/", "/boot/", "/sys/", "/proc/", "/usr/", "/dev/",
	"/var/lib/mysql/", "/var/lib/postgresql/", "/var/lib/redis/",
	"/root/",
}

// SensitiveFiles — any access (read or write) to these is blocked.
var SensitiveFiles = map[string]bool{
	"/etc/passwd":                  true,
	"/etc/shadow":                  true,
	"/etc/sudoers":                 true,
	"/etc/gshadow":                 true,
	"/etc/ssh/sshd_config":         true,
	"/root/.ssh/authorized_keys":   true,
	"/root/.bash_history":          true,
	"/etc/master.passwd":           true,
}

// SensitiveFilePatterns — regex patterns for sensitive files.
var SensitiveFilePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\.(key|pem|p12|pfx|jks)$`),
	regexp.MustCompile(`/\.ssh/(id_|authorized_)`),
	regexp.MustCompile(`/etc/sudoers\.d/`),
}

// DangerRule is a regex-based dangerous command pattern.
type DangerRule struct {
	ID      string
	Pattern *regexp.Regexp
	Reason  string
}

// DangerPatterns — dangerous command patterns (layer 4).
var DangerPatterns = []DangerRule{
	{ID: "DP-001", Pattern: regexp.MustCompile(`rm\s+.*-[rR].*\s+/\s*$`), Reason: "rm -rf / 删除根目录"},
	{ID: "DP-002", Pattern: regexp.MustCompile(`rm\s+.*-[rR].*\s+/\*`), Reason: "rm -rf /* 等价删根"},
	{ID: "DP-003", Pattern: regexp.MustCompile(`mkfs\.`), Reason: "格式化文件系统"},
	{ID: "DP-004", Pattern: regexp.MustCompile(`dd\s+.*of=/dev/`), Reason: "dd 直接写设备"},
	{ID: "DP-005", Pattern: regexp.MustCompile(`>\s*/dev/(sd|nvme|hd|vd)`), Reason: "重定向到块设备"},
	{ID: "DP-006", Pattern: regexp.MustCompile(`:\(\)\s*\{`), Reason: "fork bomb"},
	{ID: "DP-007", Pattern: regexp.MustCompile(`chmod\s+(-R\s+)?(0?777)\s+/`), Reason: "chmod 777 根目录"},
	{ID: "DP-008", Pattern: regexp.MustCompile(`chown\s+-R\s+.*\s+/\s*$`), Reason: "chown -R 根目录"},
	{ID: "DP-009", Pattern: regexp.MustCompile(`rm\s+-[rR]f?\s+/`), Reason: "rm -rf 危险路径"},
	{ID: "DP-010", Pattern: regexp.MustCompile(`find\s+/\s+.*-delete`), Reason: "find / -delete"},
	{ID: "DP-011", Pattern: regexp.MustCompile(`>\s*/etc/`), Reason: "重定向覆盖 /etc"},
	{ID: "DP-012", Pattern: regexp.MustCompile(`mv\s+/etc/`), Reason: "移动 /etc 下文件"},
}

// SudoAllowedPrefixes — commands that are allowed with sudo.
var SudoAllowedPrefixes = []string{
	"systemctl restart ",
	"systemctl start ",
	"systemctl stop ",
	"systemctl reload ",
	"logrotate ",
	"journalctl ",
}
