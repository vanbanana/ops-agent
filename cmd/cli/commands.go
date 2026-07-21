package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"ops-agent/internal/permission"
	"ops-agent/internal/safety"
)

// cmdChat runs the interactive REPL.
func cmdChat(app *App, args []string) {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	model := fs.String("model", "", "model/provider to use (provider id or model_id)")
	m := fs.String("m", "", "alias for --model")
	cont := fs.String("continue", "", "continue a session (use 'last' or any non-empty value)")
	c := fs.Bool("c", false, "continue the last session")
	sessionID := fs.String("session", "", "session id to continue")
	s := fs.String("s", "", "alias for --session")
	autoApprove := fs.Bool("auto-approve", false, "auto-approve all write operations")
	plan := fs.Bool("plan", false, "enable plan mode (read-only preview)")
	format := fs.String("format", "text", "output format: text|json")
	fs.Parse(args)

	if *m != "" {
		*model = *m
	}
	if *c {
		*cont = "last"
	}
	if *s != "" {
		*sessionID = *s
	}

	if err := applyModel(app, *model); err != nil {
		fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	applyPermissionMode(app, *autoApprove, *plan)

	render := newRenderer(*format)
	sid, err := app.resolveSession(*cont, *sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%s%s╔══════════════════════════════════════════╗%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s║   🖥️  Linux 运维智能体 — CLI 对话模式    ║%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s╚══════════════════════════════════════════╝%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s  LLM: %s | 权限: %s | 会话: %s%s\n\n", colorDim, app.activeModel(), app.permSvc.GetMode(), sid, colorReset)
	fmt.Printf("%s  输入 /quit 退出 | /tools 查看工具 | /mode 切换权限%s\n\n", colorDim, colorReset)

	for {
		fmt.Printf("%s%s你> %s", colorBold, colorGreen, colorReset)
		line, err := bufioReader.ReadString('\n')
		if err != nil {
			fmt.Println()
			break
		}
		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "/quit" || input == "/exit" {
			fmt.Printf("%s再见！%s\n", colorCyan, colorReset)
			break
		}
		if input == "/tools" {
			listTools(app)
			continue
		}
		if input == "/mode" {
			cyclePermissionMode(app)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		if err := app.runChat(ctx, sid, input, render); err != nil {
			fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
		}
		cancel()
	}
}

// cmdRun executes a single prompt and prints the final answer.
func cmdRun(app *App, args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	model := fs.String("model", "", "model/provider to use")
	m := fs.String("m", "", "alias for --model")
	cont := fs.String("continue", "", "continue a session")
	c := fs.Bool("c", false, "continue the last session")
	sessionID := fs.String("session", "", "session id to continue")
	s := fs.String("s", "", "alias for --session")
	autoApprove := fs.Bool("auto-approve", false, "auto-approve all write operations")
	plan := fs.Bool("plan", false, "enable plan mode")
	format := fs.String("format", "text", "output format: text|json")
	fs.Parse(args)

	if *m != "" {
		*model = *m
	}
	if *c {
		*cont = "last"
	}
	if *s != "" {
		*sessionID = *s
	}

	if err := applyModel(app, *model); err != nil {
		fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
	applyPermissionMode(app, *autoApprove, *plan)

	message := strings.Join(fs.Args(), " ")
	if strings.TrimSpace(message) == "" {
		fmt.Fprintf(os.Stderr, "%serror: no message provided%s\n", colorRed, colorReset)
		fs.Usage()
		os.Exit(1)
	}

	sid, err := app.resolveSession(*cont, *sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	render := newRenderer(*format)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if err := app.runChat(ctx, sid, message, render); err != nil {
		fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

// cmdSession manages sessions.
func cmdSession(app *App, args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "usage: ops-agent session {list|delete|export} [id]\n")
		os.Exit(1)
	}
	sub := args[0]
	switch sub {
	case "list", "ls":
		sessions := app.sessions.ListSessions()
		if len(sessions) == 0 {
			fmt.Println("No sessions.")
			return
		}
		fmt.Printf("%-30s %-20s %-25s %-25s\n", "ID", "TITLE", "CREATED", "UPDATED")
		for _, sess := range sessions {
			title := sess.Title
			if title == "" {
				title = "(untitled)"
			}
			fmt.Printf("%-30s %-20s %-25s %-25s\n", sess.ID, title, sess.CreatedAt, sess.UpdatedAt)
		}
	case "delete", "rm":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: ops-agent session delete <sessionID>\n")
			os.Exit(1)
		}
		if err := app.sessions.DeleteSession(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "%serror: %v%s\n", colorRed, err, colorReset)
			os.Exit(1)
		}
		fmt.Println("Session deleted.")
	case "export":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "usage: ops-agent session export <sessionID>\n")
			os.Exit(1)
		}
		msgs := app.sessions.GetRecentMessages(args[1], 1000)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(msgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown session subcommand: %s\n", sub)
		os.Exit(1)
	}
}

// cmdTools lists tools or shows subcommand help.
func cmdTools(app *App, args []string) {
	if len(args) == 0 || args[0] == "list" || args[0] == "ls" {
		listTools(app)
		return
	}
	fmt.Fprintf(os.Stderr, "usage: ops-agent tools list\n")
	os.Exit(1)
}

func listTools(app *App) {
	fmt.Printf("\n%s已注册工具:%s\n", colorYellow, colorReset)
	for i, t := range app.registry.List() {
		fmt.Printf("  %d. %s%s%s — %s [%s]\n", i+1, colorCyan, t.Name(), colorReset, t.Description(), t.Type())
	}
	fmt.Println()
}

// cmdProviders lists configured providers.
func cmdProviders(app *App, args []string) {
	if len(args) > 0 && args[0] != "list" && args[0] != "ls" {
		fmt.Fprintf(os.Stderr, "usage: ops-agent providers list\n")
		os.Exit(1)
	}
	providers := app.pool.GetAll()
	if len(providers) == 0 {
		fmt.Println("No providers configured.")
		return
	}
	fmt.Printf("%-20s %-20s %-30s %-20s %-10s\n", "ID", "NAME", "MODEL", "BASE_URL", "ACTIVE")
	for _, p := range providers {
		active := ""
		if p.IsActive {
			active = "*"
		}
		fmt.Printf("%-20s %-20s %-30s %-20s %-10s\n", p.ID, p.Name, p.ModelID, p.BaseURL, active)
	}
}

// cmdSafety scans a command.
func cmdSafety(app *App, args []string) {
	if len(args) > 0 && args[0] == "scan" {
		args = args[1:]
	}
	cmd := strings.Join(args, " ")
	if strings.TrimSpace(cmd) == "" {
		fmt.Fprintf(os.Stderr, "usage: ops-agent safety scan <command>\n")
		os.Exit(1)
	}
	result := safety.ValidateCommand(cmd)
	switch result.Status {
	case safety.StatusPassed:
		fmt.Printf("%s✅ PASSED%s\n", colorGreen, colorReset)
	case safety.StatusBlocked:
		fmt.Printf("%s🚫 BLOCKED [%s]: %s%s\n", colorRed, result.Reason, result.Detail, colorReset)
	case safety.StatusEscalate:
		fmt.Printf("%s⚠️  ESCALATE: %s%s\n", colorYellow, result.Detail, colorReset)
	}
}

// applyModel switches the active model if requested.
func applyModel(app *App, model string) error {
	if model == "" {
		return nil
	}
	return app.switchModel(model)
}

// applyPermissionMode configures the permission service.
func applyPermissionMode(app *App, autoApprove, plan bool) {
	switch {
	case plan:
		app.permSvc.SetMode(permission.ModePlan)
	case autoApprove:
		app.permSvc.SetMode(permission.ModeAutoApprove)
	default:
		app.permSvc.SetMode(permission.ModeDefault)
	}
}

// cyclePermissionMode rotates default -> auto_approve -> plan -> default.
func cyclePermissionMode(app *App) {
	switch app.permSvc.GetMode() {
	case permission.ModeDefault:
		app.permSvc.SetMode(permission.ModeAutoApprove)
	case permission.ModeAutoApprove:
		app.permSvc.SetMode(permission.ModePlan)
	case permission.ModePlan:
		app.permSvc.SetMode(permission.ModeDefault)
	}
	fmt.Printf("%s权限模式切换为: %s%s\n", colorYellow, app.permSvc.GetMode(), colorReset)
}
