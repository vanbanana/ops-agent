package safety

import (
	"strings"
)

// ValidationStatus is the result of command validation.
type ValidationStatus string

const (
	StatusPassed   ValidationStatus = "PASSED"
	StatusBlocked  ValidationStatus = "BLOCKED"
	StatusEscalate ValidationStatus = "ESCALATE"
)

// BlockReason categorizes why a command was blocked.
type BlockReason string

const (
	ReasonPattern BlockReason = "SAFETY_PATTERN_001"
	ReasonPath    BlockReason = "SAFETY_PATH_001"
	ReasonFile    BlockReason = "SAFETY_FILE_001"
	ReasonFlag    BlockReason = "SAFETY_FLAG_001"
)

// ValidationResult is the output of the validation pipeline.
type ValidationResult struct {
	Status  ValidationStatus
	Reason  BlockReason
	RuleID  string
	Detail  string
	Command string
	Warning string // Non-blocking warning (from flag specs or destructive patterns)
}

// ValidateCommand runs the AST-based validation pipeline.
// It parses the command using mvdan.cc/sh into an AST, decomposes complex
// commands (pipes, &&, ||, ;) into simple commands, and validates each one
// independently through the 5-layer pipeline:
//   1. Command name whitelist
//   2. Flag-level semantic check (new: CheckFlags)
//   3. ArgPrefix / ForbiddenArgs check
//   4. Protected paths / sensitive files
//   5. Danger pattern regexes
//
// If ANY sub-command is blocked, the entire command is blocked.
func ValidateCommand(cmd string) ValidationResult {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return ValidationResult{Status: StatusPassed, Command: cmd}
	}

	// Phase 0: AST parse — decompose into simple commands
	simpleCommands := ParseCommand(cmd)
	if len(simpleCommands) == 0 {
		return ValidationResult{Status: StatusPassed, Command: cmd}
	}

	// If parse returned a single command with empty Name, it means parse failed
	// → block conservatively (unparseable commands are suspicious)
	if len(simpleCommands) == 1 && simpleCommands[0].Name == "" {
		return ValidationResult{Status: StatusBlocked, Reason: ReasonPattern, Detail: "command parse failed (possibly malformed)", Command: cmd}
	}

	var combinedWarning string

	// Validate each sub-command independently
	for _, sc := range simpleCommands {
		result := validateSimpleCommand(sc, cmd)
		if result.Status == StatusBlocked {
			return result
		}
		if result.Warning != "" {
			combinedWarning = result.Warning
		}
		// Check redirections for dangerous targets
		for _, redir := range sc.Redirections {
			redirResult := checkRedirection(redir, cmd)
			if redirResult.Status == StatusBlocked {
				return redirResult
			}
		}
	}

	return ValidationResult{Status: StatusPassed, Command: cmd, Warning: combinedWarning}
}

// validateSimpleCommand runs the 5-layer pipeline on a single simple command.
func validateSimpleCommand(sc SimpleCommand, fullCmd string) ValidationResult {
	cmdName := sc.Name
	args := sc.Args

	// Strip path prefix
	if idx := strings.LastIndex(cmdName, "/"); idx >= 0 {
		cmdName = cmdName[idx+1:]
	}

	// Handle sudo
	if cmdName == "sudo" {
		if len(args) == 0 {
			return ValidationResult{Status: StatusBlocked, Reason: ReasonPattern, Detail: "bare sudo without command", Command: fullCmd}
		}
		innerCmd := strings.Join(args, " ")
		allowed := false
		for _, prefix := range SudoAllowedPrefixes {
			if strings.HasPrefix(innerCmd, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return ValidationResult{Status: StatusBlocked, Reason: ReasonPattern, Detail: "sudo command not in allowlist", Command: fullCmd}
		}
		innerResult := ValidateCommand(innerCmd)
		if innerResult.Status == StatusBlocked {
			return innerResult
		}
		return ValidationResult{Status: StatusEscalate, Detail: "sudo escalation recorded", Command: fullCmd}
	}

	return validateSimpleArgv(fullCmd, &cmdName, sc.Raw, args...)
}

// validateSimpleArgv runs layers 1-5 on a single parsed command.
// If cmdNamePtr is nil, it attempts to parse the raw string with simple split.
func validateSimpleArgv(fullCmd string, cmdNamePtr *string, raw string, extraArgs ...string) ValidationResult {
	var cmdName string
	var args []string

	if cmdNamePtr != nil {
		cmdName = *cmdNamePtr
		args = extraArgs
	} else {
		// Fallback: simple space split
		parts := strings.Fields(raw)
		if len(parts) == 0 {
			return ValidationResult{Status: StatusPassed, Command: fullCmd}
		}
		cmdName = parts[0]
		if idx := strings.LastIndex(cmdName, "/"); idx >= 0 {
			cmdName = cmdName[idx+1:]
		}
		args = parts[1:]
	}

	// Layer 1: Command whitelist
	spec, inWhitelist := CommandWhitelist[cmdName]
	if !inWhitelist {
		return ValidationResult{
			Status:  StatusBlocked,
			Reason:  ReasonPattern,
			Detail:  "command '" + cmdName + "' not in whitelist",
			Command: fullCmd,
		}
	}

	// Layer 2 (new): Flag-level semantic check
	flagBlocked, flagWarning, flagReason := CheckFlags(cmdName, args)
	if flagBlocked {
		return ValidationResult{
			Status:  StatusBlocked,
			Reason:  ReasonFlag,
			Detail:  flagReason,
			Command: fullCmd,
		}
	}

	// Layer 3: ArgPrefixWhitelist / ForbiddenArgs
	if len(spec.ArgPrefixWhitelist) > 0 && len(args) > 0 {
		firstArg := ""
		for _, a := range args {
			if !strings.HasPrefix(a, "-") {
				firstArg = a
				break
			}
		}
		if firstArg != "" {
			found := false
			for _, allowed := range spec.ArgPrefixWhitelist {
				if firstArg == allowed {
					found = true
					break
				}
			}
			if !found {
				return ValidationResult{
					Status:  StatusBlocked,
					Reason:  ReasonPattern,
					Detail:  "subcommand '" + firstArg + "' not allowed for " + cmdName,
					Command: fullCmd,
				}
			}
		}
	}

	if len(spec.ForbiddenArgs) > 0 {
		for _, arg := range args {
			for _, forbidden := range spec.ForbiddenArgs {
				if arg == forbidden {
					return ValidationResult{
						Status:  StatusBlocked,
						Reason:  ReasonPattern,
						Detail:  "forbidden argument '" + forbidden + "' for " + cmdName,
						Command: fullCmd,
					}
				}
			}
		}
	}

	// Layer 4: Protected paths / sensitive files
	if !spec.ReadOnly || spec.RequiresPathCheck {
		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			// Sensitive files (always block)
			if SensitiveFiles[arg] {
				return ValidationResult{
					Status:  StatusBlocked,
					Reason:  ReasonFile,
					Detail:  "access to sensitive file: " + arg,
					Command: fullCmd,
				}
			}
			for _, pat := range SensitiveFilePatterns {
				if pat.MatchString(arg) {
					return ValidationResult{
						Status:  StatusBlocked,
						Reason:  ReasonFile,
						Detail:  "access to sensitive file pattern: " + arg,
						Command: fullCmd,
					}
				}
			}
			// Protected paths (write operations only)
			if !spec.ReadOnly {
				for _, prefix := range ProtectedPathPrefixes {
					if strings.HasPrefix(arg, prefix) {
						return ValidationResult{
							Status:  StatusBlocked,
							Reason:  ReasonPath,
							Detail:  "write to protected path: " + arg,
							Command: fullCmd,
						}
					}
				}
			}
		}
	}

	// Layer 5: Danger pattern regexes (on full command)
	for _, dp := range DangerPatterns {
		if dp.Pattern.MatchString(fullCmd) {
			return ValidationResult{
				Status:  StatusBlocked,
				Reason:  ReasonPattern,
				RuleID:  dp.ID,
				Detail:  dp.Reason,
				Command: fullCmd,
			}
		}
	}

	return ValidationResult{Status: StatusPassed, Command: fullCmd, Warning: flagWarning}
}

// checkRedirection validates output redirections for dangerous targets.
func checkRedirection(redir string, fullCmd string) ValidationResult {
	// Extract target path from redirection (e.g. ">/etc/foo" → "/etc/foo")
	target := strings.TrimLeft(redir, ">< ")

	if target == "" {
		return ValidationResult{Status: StatusPassed, Command: fullCmd}
	}

	// Block redirections to sensitive files
	if SensitiveFiles[target] {
		return ValidationResult{
			Status:  StatusBlocked,
			Reason:  ReasonFile,
			Detail:  "redirection to sensitive file: " + target,
			Command: fullCmd,
		}
	}

	// Block redirections to protected paths
	for _, prefix := range ProtectedPathPrefixes {
		if strings.HasPrefix(target, prefix) {
			return ValidationResult{
				Status:  StatusBlocked,
				Reason:  ReasonPath,
				Detail:  "redirection to protected path: " + target,
				Command: fullCmd,
			}
		}
	}

	// Block redirections to block devices
	if strings.HasPrefix(target, "/dev/sd") || strings.HasPrefix(target, "/dev/nvme") ||
		strings.HasPrefix(target, "/dev/hd") || strings.HasPrefix(target, "/dev/vd") {
		return ValidationResult{
			Status:  StatusBlocked,
			Reason:  ReasonPattern,
			Detail:  "redirection to block device: " + target,
			Command: fullCmd,
		}
	}

	return ValidationResult{Status: StatusPassed, Command: fullCmd}
}
