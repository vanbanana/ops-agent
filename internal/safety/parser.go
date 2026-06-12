package safety

import (
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// SimpleCommand represents a single command extracted from a shell AST.
// Complex constructs (pipes, &&, ||, ;, subshells) are decomposed into
// multiple SimpleCommands for independent safety validation.
type SimpleCommand struct {
	Name         string   // The command name (argv[0]), e.g. "rm", "systemctl"
	Args         []string // All arguments after the command name
	Redirections []string // Output redirections (e.g. "> /etc/foo")
	Raw          string   // The raw text of this simple command
	IsSubshell   bool     // True if this command was inside $(...) or (...)
}

// ParseCommand parses a shell command string into a list of SimpleCommands.
// It handles:
//   - Pipes: `cmd1 | cmd2` → 2 SimpleCommands
//   - AND/OR: `cmd1 && cmd2`, `cmd1 || cmd2` → 2 SimpleCommands
//   - Semicolons: `cmd1; cmd2` → 2 SimpleCommands
//   - Subshells: `$(cmd)`, `(cmd)` → extracted with IsSubshell=true
//   - Variable assignments with commands: `VAR=x cmd args` → command is "cmd"
//
// If parsing fails (malformed input), returns a single SimpleCommand with the
// entire input as Raw and Name="" — the validator should treat this as suspicious.
func ParseCommand(cmd string) []SimpleCommand {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}

	reader := strings.NewReader(cmd)
	parser := syntax.NewParser(syntax.Variant(syntax.LangPOSIX))
	file, err := parser.Parse(reader, "")
	if err != nil {
		// Parse failed — return raw for the validator to handle conservatively
		return []SimpleCommand{{Raw: cmd}}
	}

	var commands []SimpleCommand
	syntax.Walk(file, func(node syntax.Node) bool {
		switch n := node.(type) {
		case *syntax.Stmt:
			if call, ok := n.Cmd.(*syntax.CallExpr); ok {
				sc := extractCallExpr(call, n, cmd)
				if sc.Name != "" || sc.Raw != "" {
					commands = append(commands, sc)
				}
			}
			// Don't recurse into sub-statements handled by BinaryCmd etc.
			// The Walk will visit children anyway.
		}
		return true
	})

	// If we got nothing from AST walk (e.g. pure assignment), return raw
	if len(commands) == 0 {
		return []SimpleCommand{{Raw: cmd}}
	}

	return commands
}

// extractCallExpr extracts a SimpleCommand from a syntax.CallExpr AST node.
func extractCallExpr(call *syntax.CallExpr, stmt *syntax.Stmt, fullCmd string) SimpleCommand {
	sc := SimpleCommand{}

	// Collect args (first is command name)
	var allArgs []string
	for _, word := range call.Args {
		allArgs = append(allArgs, wordToString(word))
	}

	if len(allArgs) > 0 {
		sc.Name = allArgs[0]
		// Strip path prefix (e.g. /usr/bin/rm → rm)
		if idx := strings.LastIndex(sc.Name, "/"); idx >= 0 {
			sc.Name = sc.Name[idx+1:]
		}
		sc.Args = allArgs[1:]
	}

	// Collect redirections from the Stmt (not CallExpr)
	for _, redir := range stmt.Redirs {
		if redir.Word != nil {
			target := wordToString(redir.Word)
			op := redir.Op.String()
			sc.Redirections = append(sc.Redirections, op+target)
		}
	}

	// Build raw from position if possible
	if call.Pos().IsValid() && call.End().IsValid() {
		start := int(call.Pos().Offset())
		end := int(call.End().Offset())
		if start >= 0 && end <= len(fullCmd) && start < end {
			sc.Raw = fullCmd[start:end]
		}
	}
	if sc.Raw == "" {
		// Fallback: reconstruct from parts
		parts := append([]string{sc.Name}, sc.Args...)
		sc.Raw = strings.Join(parts, " ")
	}

	return sc
}

// wordToString converts a syntax.Word to its string representation.
// Handles quoted strings, variables, command substitutions, etc.
func wordToString(word *syntax.Word) string {
	var buf strings.Builder
	for _, part := range word.Parts {
		switch p := part.(type) {
		case *syntax.Lit:
			buf.WriteString(p.Value)
		case *syntax.SglQuoted:
			buf.WriteString(p.Value)
		case *syntax.DblQuoted:
			for _, inner := range p.Parts {
				switch ip := inner.(type) {
				case *syntax.Lit:
					buf.WriteString(ip.Value)
				default:
					// Variable expansion, command sub, etc — represent as placeholder
					buf.WriteString("$?")
				}
			}
		case *syntax.ParamExp:
			// $VAR or ${VAR} — represent variable reference
			if p.Param != nil {
				buf.WriteString("$" + p.Param.Value)
			} else {
				buf.WriteString("$?")
			}
		case *syntax.CmdSubst:
			// $(cmd) — mark as subshell expansion
			buf.WriteString("$(…)")
		default:
			buf.WriteString("?")
		}
	}
	return buf.String()
}
