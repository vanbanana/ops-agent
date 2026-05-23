package tools

import (
	"context"
	"testing"
)

// --- Task 12.7: truncate_log_file 路径不在 /var/log/ → 拦截 ---

func TestTruncateLogPathRestriction(t *testing.T) {
	tool := &TruncateLogTool{}

	// These should be BLOCKED (path not in /var/log/)
	blockedPaths := []string{
		"/tmp/secret.txt",
		"/etc/passwd",
		"/var/log/../etc/shadow",
		"/home/user/file.log",
	}
	for _, path := range blockedPaths {
		result, _ := tool.Execute(context.Background(), map[string]any{"path": path})
		if result.Error == "" {
			t.Errorf("expected %q to be blocked, but got no error", path)
		}
		if result.Error != "" && !containsAny(result.Error, "不在", "拒绝", "..") {
			t.Errorf("expected path validation error for %q, got: %s", path, result.Error)
		}
	}

	// These should PASS validation (may fail on execution since file doesn't exist, but that's OK)
	allowedPaths := []string{
		"/var/log/app.log",
		"/var/log/nginx/access.log",
	}
	for _, path := range allowedPaths {
		result, _ := tool.Execute(context.Background(), map[string]any{"path": path})
		// If error is about path restriction, that's a bug. Execution errors (file not found) are fine.
		if result.Error != "" && containsAny(result.Error, "不在", "拒绝") {
			t.Errorf("path %q should pass validation but was rejected: %s", path, result.Error)
		}
	}

	t.Log("✅ truncate_log_file correctly restricts paths to /var/log/")
}

// --- Task 12.8: kill -9 pid → 拦截 (禁止 KILL 信号) ---

func TestKillProcessSignalRestriction(t *testing.T) {
	tool := &KillProcessTool{}

	// KILL (signal 9) should be rejected
	blockedSignals := []string{"KILL", "kill", "9", "STOP", "ABRT"}
	for _, sig := range blockedSignals {
		result, _ := tool.Execute(context.Background(), map[string]any{"pid": float64(1234), "signal": sig})
		if result.Error == "" {
			t.Errorf("expected signal %q to be blocked, but was allowed", sig)
		}
	}

	// TERM should be allowed (but will fail since PID likely doesn't exist - that's OK)
	result, _ := tool.Execute(context.Background(), map[string]any{"pid": float64(999999), "signal": "TERM"})
	// Error should be about process not found, NOT about signal being blocked
	if result.Error != "" && contains(result.Error, "不允许") {
		t.Errorf("TERM should not be blocked, got: %s", result.Error)
	}

	// PID ≤ 1 should be rejected regardless of signal
	result, _ = tool.Execute(context.Background(), map[string]any{"pid": float64(1), "signal": "TERM"})
	if result.Error == "" {
		t.Error("expected PID 1 to be rejected")
	}

	t.Log("✅ kill_process correctly blocks dangerous signals and PID ≤ 1")
}

// --- Task 12.9: service_control unit 名格式非法 → 拦截 ---

func TestServiceControlUnitValidation(t *testing.T) {
	tool := &ServiceControlTool{}

	invalidUnits := []string{
		"nginx",             // missing .service
		"../etc.service",   // path traversal
		"; rm -rf /",       // injection
		"",                 // empty
		"nginx service",    // space
		"nginx\t.service",  // tab
	}

	for _, unit := range invalidUnits {
		result, _ := tool.Execute(context.Background(), map[string]any{"action": "restart", "unit": unit})
		if result.Error == "" {
			t.Errorf("expected unit %q to be rejected, but was allowed", unit)
		}
	}

	// Valid units (will fail because systemctl isn't available on macOS, but won't fail on validation)
	validUnits := []string{"nginx.service", "sshd.service", "docker.service"}
	for _, unit := range validUnits {
		result, _ := tool.Execute(context.Background(), map[string]any{"action": "restart", "unit": unit})
		// On macOS: systemctl doesn't exist, so error is about execution, not validation
		if result.Error != "" && containsAny(result.Error, "invalid", "格式") {
			t.Errorf("unit %q should pass validation but got: %s", unit, result.Error)
		}
	}

	// Invalid action
	result, _ := tool.Execute(context.Background(), map[string]any{"action": "destroy", "unit": "nginx.service"})
	if result.Error == "" {
		t.Error("expected invalid action 'destroy' to be rejected")
	}

	t.Log("✅ service_control correctly validates unit name format and action")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAny(s, substr))
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
	}
	return false
}
