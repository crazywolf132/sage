// Package app provides the core application logic for the sage git helper tool
package app

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// SensitivityLevel indicates how critical a detected pattern is
type SensitivityLevel int

const (
	// Critical patterns should never be committed (e.g., private keys)
	Critical SensitivityLevel = iota
	// High sensitivity patterns require explicit confirmation (e.g., passwords)
	High
	// Medium sensitivity patterns show a warning (e.g., potential secrets)
	Medium
)

// SensitivePattern represents a pattern to detect in changes
type SensitivePattern struct {
	Pattern  string
	Level    SensitivityLevel
	Message  string
	Category string
}

// sensitivePatterns contains regex patterns for common sensitive data
var sensitivePatterns = []SensitivePattern{
	// Critical - Never allow these
	{
		Pattern:  `(?i)-----BEGIN\s+(?:RSA|DSA|EC|OPENSSH)\s+PRIVATE\s+KEY-----`,
		Level:    Critical,
		Message:  "Private key detected",
		Category: "Keys",
	},
	{
		Pattern:  `(?i)(?:ssh-rsa|ssh-dsa|ssh-ed25519)\s+AAAA[0-9A-Za-z+/]+[=]{0,3}`,
		Level:    Critical,
		Message:  "SSH private key detected",
		Category: "Keys",
	},

	// High - Require explicit confirmation
	{
		Pattern:  `(?i)password\s*[:=]\s*['"][^'"]+['"]`,
		Level:    High,
		Message:  "Password in plaintext",
		Category: "Credentials",
	},
	{
		Pattern:  `(?i)secret\s*[:=]\s*['"][^'"]+['"]`,
		Level:    High,
		Message:  "Secret in plaintext",
		Category: "Credentials",
	},
	{
		Pattern:  `(?i)api[_-]?key\s*[:=]\s*['"][^'"]+['"]`,
		Level:    High,
		Message:  "API key in plaintext",
		Category: "Credentials",
	},
	{
		Pattern:  `(?i)access[_-]?token\s*[:=]\s*['"][^'"]+['"]`,
		Level:    High,
		Message:  "Access token in plaintext",
		Category: "Credentials",
	},

	// Medium - Show warnings
	{
		Pattern:  `(?i)bearer\s+[a-zA-Z0-9\-\._~\+\/]+=*`,
		Level:    Medium,
		Message:  "Possible Bearer token",
		Category: "Tokens",
	},
	{
		Pattern:  `(?i)aws[_-]?(?:access|secret|key)`,
		Level:    Medium,
		Message:  "Possible AWS credential",
		Category: "Cloud",
	},
	{
		Pattern:  `(?i)(?:mongodb|postgres|mysql|redis|rabbitmq).*=.+`,
		Level:    Medium,
		Message:  "Possible database connection string",
		Category: "Database",
	},
}

// Finding represents a detected sensitive data match
type Finding struct {
	Pattern  *SensitivePattern
	Line     string
	Category string
}

// CommitOptions defines the parameters for creating a git commit.
// It provides configuration for AI-assisted commit messages, conventional commits,
// and post-commit actions like pushing.
type CommitOptions struct {
	// Message is the commit message to use. If empty, will prompt or use AI
	Message string
	// AllowEmpty enables creating commits with no changes
	AllowEmpty bool
	// PushAfterCommit determines if the commit should be pushed immediately
	PushAfterCommit bool
	// UseConventional enables conventional commit format (type: message)
	UseConventional bool
	// UseAI enables AI-generated commit messages when no message is provided
	UseAI bool
	// AutoAccept skips the confirmation prompt for AI-generated messages
	AutoAccept bool
	// ChangeType allows overriding the commit type (feat, fix, etc)
	ChangeType string
	// Amend determines if the last commit should be amended
	Amend bool
}

// CommitResult contains the outcome of a commit operation.
type CommitResult struct {
	// ActualMessage is the final commit message used
	ActualMessage string
	// Pushed indicates if the commit was pushed to remote
	Pushed bool
}

// changeCommitType modifies a commit message to use a different conventional commit type.
// It handles both simple messages and those with scopes - e.g., feat(scope): message.
func changeCommitType(msg, newType string) string {
	// If message doesn't follow conventional format, prepend the type
	if !strings.Contains(msg, ":") {
		return fmt.Sprintf("%s: %s", newType, msg)
	}
	// Split into type and message parts
	parts := strings.SplitN(msg, ":", 2)
	typeScope := parts[0]
	message := strings.TrimSpace(parts[1])

	// Check if there's a scope
	if strings.Contains(typeScope, "(") {
		scope := strings.Split(typeScope, "(")[1]
		scope = strings.TrimRight(scope, ")")
		return fmt.Sprintf("%s(%s): %s", newType, scope, message)
	}

	// Handle messages without scope
	return fmt.Sprintf("%s: %s", newType, message)
}

// detectSensitiveData checks for potential sensitive data in the changes
func detectSensitiveData(g git.Service) ([]Finding, error) {
	// Get staged changes
	diff, err := g.StagedDiff()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged changes: %w", err)
	}

	var findings []Finding
	for _, pattern := range sensitivePatterns {
		matches, err := g.GrepDiff(diff, pattern.Pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			findings = append(findings, Finding{
				Pattern:  &pattern,
				Line:     match,
				Category: pattern.Category,
			})
		}
	}

	return findings, nil
}

// formatFindings formats the findings into a user-friendly message
func formatFindings(findings []Finding) string {
	if len(findings) == 0 {
		return ""
	}

	var critical, high, medium []string
	for _, f := range findings {
		msg := fmt.Sprintf("- %s: %s", f.Pattern.Message, f.Line)
		switch f.Pattern.Level {
		case Critical:
			critical = append(critical, msg)
		case High:
			high = append(high, msg)
		case Medium:
			medium = append(medium, msg)
		}
	}

	var result strings.Builder
	result.WriteString("Sensitive data detected in changes:\n\n")

	if len(critical) > 0 {
		result.WriteString("🚨 CRITICAL - Must be removed:\n")
		result.WriteString(strings.Join(critical, "\n"))
		result.WriteString("\n\n")
	}

	if len(high) > 0 {
		result.WriteString("⚠️  HIGH RISK - Should be removed:\n")
		result.WriteString(strings.Join(high, "\n"))
		result.WriteString("\n\n")
	}

	if len(medium) > 0 {
		result.WriteString("ℹ️  MEDIUM RISK - Please review:\n")
		result.WriteString(strings.Join(medium, "\n"))
		result.WriteString("\n")
	}

	result.WriteString("\nRecommendations:\n")
	result.WriteString("1. Remove sensitive data from the changes\n")
	result.WriteString("2. Consider using environment variables or a secrets manager\n")
	result.WriteString("3. If these are test credentials, use placeholder values\n")

	return result.String()
}

// Commit implements our simplified commit pipeline.
// It automatically stages every file, then (if no message is provided)
// uses AI (if enabled) to generate a commit message.
func Commit(g git.Service, opts CommitOptions) (CommitResult, error) {
	result := CommitResult{}

	// Verify we're in a Git repository.
	isRepo, err := g.IsRepo()
	if err != nil || !isRepo {
		return result, fmt.Errorf("not a git repository")
	}

	// Check for sensitive data before proceeding
	findings, err := detectSensitiveData(g)
	if err != nil {
		return result, err
	}
	if len(findings) > 0 {
		// Block commit if there are any critical findings
		for _, f := range findings {
			if f.Pattern.Level == Critical {
				return result, fmt.Errorf(formatFindings(findings))
			}
		}

		// For high/medium findings, show warning but allow commit
		ui.Warning(formatFindings(findings))
		if !opts.AllowEmpty {
			var proceed bool
			prompt := &survey.Confirm{
				Message: "Do you want to proceed with the commit despite the warnings?",
				Default: false,
			}
			if err := survey.AskOne(prompt, &proceed); err != nil {
				return result, err
			}
			if !proceed {
				return result, fmt.Errorf("commit cancelled due to sensitive data")
			}
		}
	}

	// Get the current status.
	status, err := g.StatusPorcelain()
	if err != nil {
		return result, fmt.Errorf("failed to get status: %w", err)
	}
	if strings.TrimSpace(status) == "" && !opts.AllowEmpty {
		return result, fmt.Errorf("no changes to commit")
	}

	// Get current branch for metadata
	branch, err := g.CurrentBranch()
	if err != nil {
		return result, fmt.Errorf("failed to get current branch: %w", err)
	}

	// Get list of changed files for metadata
	var files []string
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if line == "" {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[1]
		}
		files = append(files, path)
	}

	// If no commit message was provided...
	if opts.Message == "" {
		if opts.UseAI {
			diff, err := g.GetDiff()
			if err != nil {
				return result, fmt.Errorf("failed to get diff: %w", err)
			}
			client := ai.NewClient("", ai.NewConfigAdapter(config.Get))
			if client.APIKey == "" {
				return result, fmt.Errorf("AI features require an OpenAI API key")
			}
			aiMsg, err := client.GenerateCommitMessage(diff)
			if err != nil {
				return result, fmt.Errorf("failed to generate AI commit message: %w", err)
			}
			fmt.Printf("Generated commit message: %q\n", aiMsg)
			if opts.AutoAccept {
				// Automatically accept the AI message
				opts.Message = aiMsg
			} else {
				// Otherwise, prompt the user.
				var choice string
				err = survey.AskOne(&survey.Select{
					Message: "Choose an option:",
					Options: []string{"Accept", "Change type", "Enter manually"},
				}, &choice)
				if err != nil {
					return result, err
				}

				switch choice {
				case "Accept":
					opts.Message = aiMsg
					break
				case "Change type":
					newType := ""
					err = survey.AskOne(&survey.Select{
						Message: "Select new commit type:",
						Options: []string{
							"feat", "fix", "docs", "style", "test", "ci", "refactor", "perf", "chore",
						},
					}, &newType)
					if err != nil {
						return result, err
					}
					opts.Message = changeCommitType(aiMsg, newType)
					break
				case "Enter manually":
					msg, scope, ctype, err := ui.AskCommitMessage(opts.UseConventional)
					if err != nil {
						return result, err
					}
					if opts.UseConventional {
						if scope != "" {
							opts.Message = fmt.Sprintf("%s(%s): %s", ctype, scope, msg)
						} else {
							opts.Message = fmt.Sprintf("%s: %s", ctype, msg)
						}
					} else {
						opts.Message = msg
					}
				}
			}
		} else {
			msg, scope, ctype, err := ui.AskCommitMessage(opts.UseConventional)
			if err != nil {
				return result, err
			}
			if opts.UseConventional {
				if scope != "" {
					opts.Message = fmt.Sprintf("%s(%s): %s", ctype, scope, msg)
				} else {
					opts.Message = fmt.Sprintf("%s: %s", ctype, msg)
				}
			} else {
				opts.Message = msg
			}
		}
	}

	// If a commit type change is requested, update the message.
	if opts.ChangeType != "" {
		opts.Message = changeCommitType(opts.Message, opts.ChangeType)
	}

	// Stage everything (we no longer exclude .sage/).
	if err := g.StageAll(); err != nil {
		return result, err
	}

	// Create the commit with the final message and options
	if opts.Amend {
		if shellGit, ok := g.(*git.ShellGit); ok {
			if err := shellGit.CommitAmend(opts.Message, opts.AllowEmpty, true); err != nil {
				return result, fmt.Errorf("failed to amend commit: %w", err)
			}
		} else {
			return result, fmt.Errorf("amend flag is not supported for this git implementation")
		}
	} else {
		if err := g.Commit(opts.Message, opts.AllowEmpty, true); err != nil {
			return result, fmt.Errorf("failed to commit: %w", err)
		}
	}

	// Record the operation in undo history
	if err := RecordOperation(g, "commit", opts.Message, "git commit", "commit", files, branch, opts.Message, false, ""); err != nil {
		ui.Warning("Failed to record operation in undo history")
	}

	result.ActualMessage = opts.Message

	// Push the commit to remote if requested
	if opts.PushAfterCommit {
		// Push changes to remote repository
		if err := g.Push(branch, false); err != nil {
			return result, err
		}
		result.Pushed = true
	}

	return result, nil
}
