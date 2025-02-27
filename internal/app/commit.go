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
	// OnlyStaged determines if only staged changes should be committed
	OnlyStaged bool
	// Interactive determines if the user should interactively select files
	Interactive bool
}

// CommitResult contains the outcome of a commit operation.
type CommitResult struct {
	// ActualMessage is the final commit message used
	ActualMessage string
	// Pushed indicates if the commit was pushed to remote
	Pushed bool
	// Stats provides statistics about the committed files
	Stats CommitResultStats
}

// CommitResultStats represents the statistics of staged and unstaged files
type CommitResultStats struct {
	StagedAdded    int
	StagedModified int
	StagedDeleted  int
	TotalStaged    int
	TotalUnstaged  int
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
// It automatically stages every file (unless onlyStaged is true), then (if no message is provided)
// uses AI (if enabled) to generate a commit message.
func Commit(g git.Service, opts CommitOptions) (CommitResult, error) {
	var result CommitResult

	// Check if only-staged should be the default from config
	if !opts.OnlyStaged && !opts.Interactive {
		// Check if user has configured a default behavior for commit
		onlyStagedDefault := config.Get("commit.only_staged_default", false) == "true"
		if onlyStagedDefault {
			opts.OnlyStaged = true
		}
	}

	// If amend is set, check that there is a previous commit.
	if opts.Amend {
		// Get the last commit message.
		lastMessageOutput, err := g.Run("log", "--format=%B", "-n", "1")
		if err != nil {
			return result, fmt.Errorf("failed to get last commit message: %w", err)
		}
		// If no message is set, use the last commit message by default.
		if opts.Message == "" {
			opts.Message = strings.TrimSpace(lastMessageOutput)
		}
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

	// Check if there are changes to commit
	hasUnstagedChanges := false
	hasStagedChanges := false

	// Parse files from status to track both staged and unstaged
	var unstagedFiles []string
	var stagedFiles []string
	stats := CommitResultStats{}

	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if line == "" {
			continue
		}
		statusCode := line[:2]
		filePath := strings.TrimSpace(line[3:])
		x, y := statusCode[0], statusCode[1]

		// Check if file is staged (X is not space or ?)
		if x != ' ' && x != '?' {
			hasStagedChanges = true
			stagedFiles = append(stagedFiles, filePath)

			// Track staged file stats
			switch x {
			case 'A':
				stats.StagedAdded++
			case 'M':
				stats.StagedModified++
			case 'D':
				stats.StagedDeleted++
			}
			stats.TotalStaged++
		}

		// Check if file has unstaged changes (Y is not space)
		if y != ' ' {
			hasUnstagedChanges = true
			unstagedFiles = append(unstagedFiles, filePath)
			stats.TotalUnstaged++
		}
	}

	// If no changes (staged or unstaged) and empty commits not allowed
	if !hasStagedChanges && !hasUnstagedChanges && !opts.AllowEmpty {
		return result, fmt.Errorf("no changes to commit")
	}

	// Interactive mode: Let the user select which files to stage
	if opts.Interactive && hasUnstagedChanges {
		fmt.Println(ui.Bold("Select files to stage for this commit:"))

		// Create checkboxes for all unstaged files
		fileOptions := make([]string, 0, len(unstagedFiles))
		defaultSelected := make([]string, 0)

		// Include all files, with special handling for dot directories
		for _, file := range unstagedFiles {
			fileOptions = append(fileOptions, file)
			// Auto-select .github files to ensure they're visible and included by default
			if strings.Contains(file, ".github/") {
				defaultSelected = append(defaultSelected, file)
			}
		}

		// If there are no unstaged files but the interactive flag is set
		if len(fileOptions) == 0 {
			fmt.Println(ui.Yellow("No unstaged files to select. Proceeding with already staged files."))
			opts.OnlyStaged = true
		} else {
			// Ask user which files to stage
			var selectedFiles []string
			prompt := &survey.MultiSelect{
				Message: "Select files to stage:",
				Options: fileOptions,
				Default: defaultSelected,
			}

			if err := survey.AskOne(prompt, &selectedFiles); err != nil {
				return result, fmt.Errorf("canceled: %w", err)
			}

			// If no files selected, use only-staged mode if there are already staged files
			if len(selectedFiles) == 0 {
				if hasStagedChanges {
					fmt.Println(ui.Yellow("No files selected. Proceeding with already staged files."))
					opts.OnlyStaged = true
				} else {
					return result, fmt.Errorf("no files selected to commit")
				}
			} else {
				// Stage the selected files - use Run method to stage each file
				for _, file := range selectedFiles {
					if _, err := g.Run("add", file); err != nil {
						return result, fmt.Errorf("failed to stage %s: %w", file, err)
					}
				}

				fmt.Printf("%s Staged %d file(s)\n", ui.Green("✓"), len(selectedFiles))

				// Now we only want to commit what we've just staged
				opts.OnlyStaged = true
				hasStagedChanges = true
			}
		}
	}

	// Smart mode: If there are staged changes but --only-staged flag wasn't explicitly set,
	// and there are also unstaged changes, ask the user what they want to do
	if !opts.OnlyStaged && hasStagedChanges && hasUnstagedChanges && !opts.Interactive {
		var choice string
		prompt := &survey.Select{
			Message: "You have both staged and unstaged changes. What would you like to do?",
			Options: []string{
				"Commit only staged changes",
				"Stage all changes and commit everything",
				"View what's staged vs. unstaged before deciding",
			},
			Default: "Commit only staged changes",
		}

		if err := survey.AskOne(prompt, &choice); err != nil {
			return result, fmt.Errorf("canceled: %w", err)
		}

		switch choice {
		case "Commit only staged changes":
			opts.OnlyStaged = true
		case "View what's staged vs. unstaged before deciding":
			// Show a summary of staged vs unstaged changes
			fmt.Println("\n" + ui.Bold("Staged changes (will be committed):"))
			stagedDiff, err := g.StagedDiff()
			if err == nil && stagedDiff != "" {
				// Find up to 5 files to show as a summary
				var stagedFiles []string
				for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
					if line == "" {
						continue
					}
					statusCode := line[:2]
					path := strings.TrimSpace(line[3:])
					x := statusCode[0]
					if x != ' ' && x != '?' {
						if strings.Contains(path, " -> ") {
							parts := strings.Split(path, " -> ")
							path = parts[1]
						}
						stagedFiles = append(stagedFiles, ui.Green("+ "+path))
						if len(stagedFiles) >= 5 {
							break
						}
					}
				}
				fmt.Println(strings.Join(stagedFiles, "\n"))
				if len(stagedFiles) == 5 {
					fmt.Println("... and more")
				}
			} else {
				fmt.Println("No staged changes")
			}

			fmt.Println("\n" + ui.Bold("Unstaged changes (will NOT be committed):"))
			var unstagedFiles []string
			for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
				if line == "" {
					continue
				}
				statusCode := line[:2]
				path := strings.TrimSpace(line[3:])
				x, y := statusCode[0], statusCode[1]
				if (x == ' ' || x == '?') && y != ' ' {
					if strings.Contains(path, " -> ") {
						parts := strings.Split(path, " -> ")
						path = parts[1]
					}
					unstagedFiles = append(unstagedFiles, ui.Yellow("? "+path))
					if len(unstagedFiles) >= 5 {
						break
					}
				}
			}
			fmt.Println(strings.Join(unstagedFiles, "\n"))
			if len(unstagedFiles) == 5 {
				fmt.Println("... and more")
			}

			// Now ask again after showing the diff
			var secondChoice string
			prompt := &survey.Select{
				Message: "What would you like to do?",
				Options: []string{
					"Commit only staged changes",
					"Stage all changes and commit everything",
					"Cancel and go back to staging",
				},
				Default: "Commit only staged changes",
			}

			if err := survey.AskOne(prompt, &secondChoice); err != nil {
				return result, fmt.Errorf("canceled: %w", err)
			}

			switch secondChoice {
			case "Commit only staged changes":
				opts.OnlyStaged = true
			case "Cancel and go back to staging":
				return result, fmt.Errorf("commit canceled - use 'sage stage' to select files to stage")
			}
		}
	}

	// If onlyStaged is true but there are no staged changes
	if opts.OnlyStaged && !hasStagedChanges && !opts.AllowEmpty {
		return result, fmt.Errorf("no staged changes to commit (use 'sage stage' first or remove --only-staged flag)")
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

	// Stage all changes if not using only staged changes
	if !opts.OnlyStaged {
		if err := g.StageAll(); err != nil {
			return result, err
		}
	}

	// Create the commit with the final message and options
	if opts.Amend {
		if shellGit, ok := g.(*git.ShellGit); ok {
			if err := shellGit.CommitAmend(opts.Message, opts.AllowEmpty, !opts.OnlyStaged); err != nil {
				return result, fmt.Errorf("failed to amend commit: %w", err)
			}
		} else {
			return result, fmt.Errorf("amend flag is not supported for this git implementation")
		}
	} else {
		if err := g.Commit(opts.Message, opts.AllowEmpty, !opts.OnlyStaged); err != nil {
			return result, fmt.Errorf("failed to commit: %w", err)
		}
	}

	// Record the operation in undo history
	if err := RecordOperation(g, "commit", opts.Message, "git commit", "commit", files, branch, opts.Message, false, ""); err != nil {
		ui.Warning("Failed to record operation in undo history")
	}

	result.ActualMessage = opts.Message
	result.Stats = stats

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
