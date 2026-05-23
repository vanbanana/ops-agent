package safety

import (
	"strings"

	"github.com/google/shlex"
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
)

// ValidationResult is the output of the five-layer validation pipeline.
type ValidationResult struct {
	Status  ValidationStatus
	Reason  BlockReason
	RuleID  string
	Detail  string
	Command string
}

// ValidateCommand runs the five-layer validation pipeline.
func ValidateCommand(cmd string) ValidationResult {
	cmd = strings.TrimSpace(cmd)

	// Layer 0: Parse
	argv, err := shlex.Split(cmd)
	if err != nil {
		return ValidationResult{
			Status:  StatusBlocked,
			Reason:  ReasonPattern,
			Detail:  "command cannot be parsed: " + err.Error(),
			Command: cmd,
		}
	}

	// Empty command passes
	if len(argv) == 0 {
		return ValidationResult{Status: StatusPassed, Command: cmd}
	}

	cmdName := argv[0]

	// Strip leading path (e.g. /bin/rm → rm)
	if idx := strings.LastIndex(cmdName, "/"); idx >= 0 {
		cmdName = cmdName[idx+1:]
	}

	// Check for sudo
	isSudo := cmdName == "sudo"
	if isSudo {
		if len(argv) < 2 {
			return ValidationResult{Status: StatusBlocked, Reason: ReasonPattern, Detail: "bare sudo without command", Command: cmd}
		}
		// Validate the inner command
		innerCmd := strings.Join(argv[1:], " ")
		// Check sudo allowlist
		allowed := false
		for _, prefix := range SudoAllowedPrefixes {
			if strings.HasPrefix(innerCmd, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return ValidationResult{Status: StatusBlocked, Reason: ReasonPattern, Detail: "sudo command not in allowlist", Command: cmd}
		}
		// For allowed sudo commands, validate the inner part
		innerResult := ValidateCommand(innerCmd)
		if innerResult.Status == StatusBlocked {
			return innerResult
		}
		return ValidationResult{Status: StatusEscalate, Detail: "sudo escalation recorded", Command: cmd}
	}

	// Layer 1: Command whitelist
	spec, inWhitelist := CommandWhitelist[cmdName]
	if !inWhitelist {
		return ValidationResult{
			Status:  StatusBlocked,
			Reason:  ReasonPattern,
			Detail:  "command '" + cmdName + "' not in whitelist",
			Command: cmd,
		}
	}

	// Check ArgPrefixWhitelist (first non-flag arg)
	if len(spec.ArgPrefixWhitelist) > 0 && len(argv) > 1 {
		firstArg := ""
		for _, a := range argv[1:] {
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
					Command: cmd,
				}
			}
		}
	}

	// Check ForbiddenArgs
	if len(spec.ForbiddenArgs) > 0 {
		for _, arg := range argv[1:] {
			for _, forbidden := range spec.ForbiddenArgs {
				if arg == forbidden {
					return ValidationResult{
						Status:  StatusBlocked,
						Reason:  ReasonPattern,
						Detail:  "forbidden argument '" + forbidden + "' for " + cmdName,
						Command: cmd,
					}
				}
			}
		}
	}

	// Layer 2: Protected paths (for write operations)
	if !spec.ReadOnly || spec.RequiresPathCheck {
		for _, arg := range argv[1:] {
			if strings.HasPrefix(arg, "-") {
				continue
			}
			// Layer 3: Sensitive files (always block)
			if SensitiveFiles[arg] {
				return ValidationResult{
					Status:  StatusBlocked,
					Reason:  ReasonFile,
					Detail:  "access to sensitive file: " + arg,
					Command: cmd,
				}
			}
			for _, pat := range SensitiveFilePatterns {
				if pat.MatchString(arg) {
					return ValidationResult{
						Status:  StatusBlocked,
						Reason:  ReasonFile,
						Detail:  "access to sensitive file pattern: " + arg,
						Command: cmd,
					}
				}
			}

			// Protected paths only block write operations
			if !spec.ReadOnly {
				for _, prefix := range ProtectedPathPrefixes {
					if strings.HasPrefix(arg, prefix) {
						return ValidationResult{
							Status:  StatusBlocked,
							Reason:  ReasonPath,
							Detail:  "write to protected path: " + arg,
							Command: cmd,
						}
					}
				}
			}
		}
	}

	// Layer 4: Danger patterns (full command string)
	for _, dp := range DangerPatterns {
		if dp.Pattern.MatchString(cmd) {
			return ValidationResult{
				Status:  StatusBlocked,
				Reason:  ReasonPattern,
				RuleID:  dp.ID,
				Detail:  dp.Reason,
				Command: cmd,
			}
		}
	}

	return ValidationResult{Status: StatusPassed, Command: cmd}
}
