package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"ops-agent/internal/safety"
)

// TerminalHandler provides a WebSocket-like terminal endpoint using SSE + POST.
// We use SSE for output streaming and POST for command input (simpler than WS).

// POST /api/v1/terminal/exec — execute a command and stream output
func HandleTerminalExec(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "invalid request"})
		return
	}

	command := strings.TrimSpace(req.Command)
	if command == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"code": 400, "error": "command is required"})
		return
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 120 {
		timeout = 120
	}

	// Safety validation
	result := safety.ValidateCommand(command)
	if result.Status == safety.StatusBlocked {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"command":   command,
				"output":    "",
				"error":     fmt.Sprintf("blocked: %s — %s", result.Reason, result.Detail),
				"exit_code": -1,
				"blocked":   true,
			},
		})
		return
	}

	// Execute with streaming output collection
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		writeExecError(w, command, err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		writeExecError(w, command, err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		writeExecError(w, command, err.Error())
		return
	}

	// Collect output from both streams
	var output strings.Builder
	var mu sync.Mutex
	var wg sync.WaitGroup

	collect := func(scanner *bufio.Scanner) {
		defer wg.Done()
		for scanner.Scan() {
			mu.Lock()
			output.WriteString(scanner.Text())
			output.WriteByte('\n')
			mu.Unlock()
		}
	}

	wg.Add(2)
	go collect(bufio.NewScanner(stdout))
	go collect(bufio.NewScanner(stderr))
	wg.Wait()

	exitCode := 0
	waitErr := cmd.Wait()
	if waitErr != nil {
		if ctx.Err() != nil {
			writeExecResult(w, command, output.String()+"[timeout]\n", 124)
			return
		}
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Truncate if too long
	out := output.String()
	if len(out) > 50000 {
		out = out[:25000] + "\n... [truncated] ...\n" + out[len(out)-25000:]
	}

	writeExecResult(w, command, out, exitCode)
}

func writeExecResult(w http.ResponseWriter, command, output string, exitCode int) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"code": 0,
		"data": map[string]any{
			"command":   command,
			"output":    output,
			"exit_code": exitCode,
			"blocked":   false,
		},
	})
}

func writeExecError(w http.ResponseWriter, command, errMsg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"code": 0,
		"data": map[string]any{
			"command":   command,
			"output":    "",
			"error":     errMsg,
			"exit_code": -1,
			"blocked":   false,
		},
	})
}
