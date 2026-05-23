package api

import (
	"testing"
)

// --- Path validation tests (security critical) ---

func TestValidatePathBlocked(t *testing.T) {
	blocked := []string{
		"/proc",
		"/proc/1/status",
		"/sys",
		"/sys/class/net",
		"/dev",
		"/dev/sda",
		"/boot",
		"/boot/grub",
		"/etc/shadow",
		"/etc/gshadow",
		"/etc/sudoers",
	}

	for _, path := range blocked {
		_, err := validatePath(path)
		if err == nil {
			t.Errorf("expected %q to be blocked, but was allowed", path)
		}
	}
	t.Logf("✅ All %d dangerous paths correctly blocked", len(blocked))
}

func TestValidatePathAllowed(t *testing.T) {
	allowed := []string{
		"/",
		"/home",
		"/var/log",
		"/var/log/nginx/access.log",
		"/tmp",
		"/opt",
		"/usr/local/bin",
		"/etc/nginx",
		"/etc/hosts",
	}

	for _, path := range allowed {
		result, err := validatePath(path)
		if err != nil {
			t.Errorf("expected %q to be allowed, got error: %v", path, err)
		}
		if result == "" {
			t.Errorf("expected non-empty cleaned path for %q", path)
		}
	}
	t.Logf("✅ All %d safe paths correctly allowed", len(allowed))
}

func TestValidatePathTraversal(t *testing.T) {
	traversal := []string{
		"/var/log/../../etc/shadow",
		"/../../../etc/passwd",
		"/tmp/../proc/1/status",
	}

	for _, path := range traversal {
		_, err := validatePath(path)
		if err == nil {
			t.Errorf("expected traversal path %q to be blocked", path)
		}
	}
	t.Logf("✅ Path traversal attempts correctly blocked")
}

func TestValidatePathCleaning(t *testing.T) {
	// Relative paths should get prefixed with /
	result, err := validatePath("var/log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "/var/log" {
		t.Fatalf("expected /var/log, got %q", result)
	}
}
