package tools

import (
	"context"
	"strings"
	"testing"
)

// Task 8.14: fixture-based output parsing tests for each probe
func TestProbeDiskExecutes(t *testing.T) {
	probe := NewProbeDisk()
	result, err := probe.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("probe_disk failed: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("probe_disk returned empty summary")
	}
	// Should contain filesystem info
	if !strings.Contains(result.Summary, "/") {
		t.Fatal("probe_disk output should contain mount points")
	}
}

func TestProbeProcessExecutes(t *testing.T) {
	probe := NewProbeProcess()
	result, err := probe.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("probe_process failed: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("probe_process returned empty summary")
	}
}

// Task 8.15: probe_process truncation test
func TestProbeProcessTruncation(t *testing.T) {
	probe := NewProbeProcess()
	result, err := probe.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("probe_process failed: %v", err)
	}
	lines := strings.Split(result.Summary, "\n")
	// Should have header + at most 10 data lines + summary line
	// Total should not exceed ~12 lines regardless of system process count
	if len(lines) > 15 {
		t.Fatalf("probe_process should truncate to ~10 processes, got %d lines", len(lines))
	}
	// Should indicate truncation if system has >10 processes
	if result.Truncated {
		if !strings.Contains(result.Summary, "共") || !strings.Contains(result.Summary, "进程") {
			t.Fatal("truncated output should contain process count summary")
		}
	}
}

func TestProbeMemoryExecutes(t *testing.T) {
	probe := NewProbeMemory()
	result, err := probe.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("probe_memory failed: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("probe_memory returned empty summary")
	}
}

func TestProbeSystemInfoExecutes(t *testing.T) {
	probe := NewProbeSystemInfo()
	result, err := probe.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("probe_system_info failed: %v", err)
	}
	if !strings.Contains(result.Summary, "System:") {
		t.Fatal("probe_system_info should contain 'System:' prefix")
	}
}

func TestProbeNetworkInterfacesExecutes(t *testing.T) {
	probe := NewProbeNetworkInterfaces()
	result, err := probe.Execute(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("probe_network_interfaces failed: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("probe_network_interfaces returned empty summary")
	}
}

// Task 8.16: command not found returns friendly error, not panic
func TestProbeLogsJournalNoJournalctl(t *testing.T) {
	// On macOS, journalctl doesn't exist — should return friendly message not panic
	probe := NewProbeLogsJournal()
	result, err := probe.Execute(context.Background(), map[string]any{"since": "1 hour ago"})
	// Should not panic and should not return a Go error
	if err != nil {
		t.Fatalf("probe_logs_journal should handle missing command gracefully, got error: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("should return a friendly message when journalctl unavailable")
	}
}

func TestProbeServiceStatusNoSystemctl(t *testing.T) {
	probe := NewProbeServiceStatus()
	result, err := probe.Execute(context.Background(), map[string]any{"service": "nginx"})
	// On macOS without systemd, should not panic
	if err != nil {
		t.Fatalf("probe_service_status should handle missing systemctl gracefully, got: %v", err)
	}
	if result.Summary == "" {
		t.Fatal("should return a friendly message")
	}
}

func TestProbeLogsFileNonexistentPath(t *testing.T) {
	probe := NewProbeLogsFile()
	_, err := probe.Execute(context.Background(), map[string]any{"path": "/nonexistent/path/file.log"})
	// Should return an error, not panic
	if err == nil {
		t.Fatal("probe_logs_file with nonexistent path should return error")
	}
}

// Registry test
func TestRegistryToolCount(t *testing.T) {
	r := NewRegistry()
	RegisterAllProbes(r)
	if len(r.List()) < 12 {
		t.Fatalf("expected >= 12 tools, got %d", len(r.List()))
	}
}

func TestRegistryDispatchUnknownTool(t *testing.T) {
	r := NewRegistry()
	RegisterAllProbes(r)
	result := r.Dispatch(context.Background(), "nonexistent_tool", nil)
	if result.Error == "" {
		t.Fatal("dispatching unknown tool should return error")
	}
}
