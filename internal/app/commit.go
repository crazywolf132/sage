// Package app provides the core application logic for the sage git helper tool
package app

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

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
	if !strings.Contains(msg, ": ") {
		return fmt.Sprintf("%s: %s", newType, msg)
	}
	// Split into type and message parts
	parts := strings.SplitN(msg, ": ", 2)
	// Handle messages with scope - e.g., feat(api): message
	if strings.Contains(parts[0], "(") {
		typeParts := strings.SplitN(parts[0], "(", 2)
		scope := "(" + typeParts[1] // includes the closing parenthesis
		return fmt.Sprintf("%s%s: %s", newType, scope, parts[1])
	}
	// Handle messages without scope
	return fmt.Sprintf("%s: %s", newType, parts[1])
}

// Commit implements our simplified commit pipeline.
// It automatically stages every file, then (if no message is provided)
// uses AI (if enabled) to generate a commit message.
func Commit(g git.Service, opts CommitOptions) (*CommitResult, error) {
	// Verify we’re in a Git repository.
	isRepo, err := g.IsRepo()
	if err != nil || !isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	// Get the current status.
	status, err := g.StatusPorcelain()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	if strings.TrimSpace(status) == "" && !opts.AllowEmpty {
		return nil, fmt.Errorf("no changes to commit")
	}

	// If no commit message was provided...
	if opts.Message == "" {
		if opts.UseAI {
			diff, err := g.GetDiff()
			if err != nil {
				return nil, fmt.Errorf("failed to get diff: %w", err)
			}
			client := ai.NewClient("")
			if client.APIKey == "" {
				return nil, fmt.Errorf("AI features require an OpenAI API key")
			}
			aiMsg, err := client.GenerateCommitMessage(diff)
			if err != nil {
				return nil, fmt.Errorf("failed to generate AI commit message: %w", err)
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
					return nil, err
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
						return nil, err
					}
					opts.Message = changeCommitType(aiMsg, newType)
					break
				case "Enter manually":
					msg, scope, ctype, err := ui.AskCommitMessage(opts.UseConventional)
					if err != nil {
						return nil, err
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
				return nil, err
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
		return nil, err
	}

	// Create the commit with the final message and options
	if err := g.Commit(opts.Message, opts.AllowEmpty, true); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}
	// Create result object with the actual message used
	res := &CommitResult{ActualMessage: opts.Message}

	// Push the commit to remote if requested
	if opts.PushAfterCommit {
		// Get current branch name for pushing
		branch, err := g.CurrentBranch()
		if err != nil {
			return nil, err
		}
		// Push changes to remote repository
		if err := g.Push(branch, false); err != nil {
			return nil, err
		}
		// Mark the result as pushed
		res.Pushed = true
	}

	// Return the commit result with message and push status
	return res, nil
}
