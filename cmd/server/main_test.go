package main

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestGoVersionMinimum(t *testing.T) {
	version := runtime.Version()
	parts := strings.Split(strings.TrimPrefix(version, "go"), ".")
	if len(parts) < 2 || parts[0] != "1" || parts[1] < "21" {
		t.Fatalf("Go version must be >= 1.21, got %s", version)
	}
}

func TestGoModDeclaresMinVersion(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("cannot read go.mod: %v", err)
	}
	if !strings.Contains(string(data), "go 1.25") {
		t.Fatalf("go.mod should declare go 1.25.x")
	}
}

func TestLoong64CrossCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cross-compile check in short mode")
	}
	out, err := exec.Command("go", "tool", "dist", "list").Output()
	if err != nil {
		t.Fatalf("go tool dist list failed: %v", err)
	}
	if !strings.Contains(string(out), "linux/loong64") {
		t.Fatal("linux/loong64 not in supported platforms")
	}
}
