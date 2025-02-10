package app

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// CommitOptions defines our commit parameters.
type CommitOptions struct {
	Message         string
	AllowEmpty      bool
	PushAfterCommit bool
	UseConventional bool // whether to prompt with a conventional commit format
	UseAI           bool // if true, generate commit message via AI if none provided
	AutoAccept      bool // if true, automatically accept AI-generated message without prompting
	ChangeType      string
}

// CommitResult contains the result of the commit.
type CommitResult struct {
	ActualMessage string
	Pushed        bool
}

func changeCommitType(msg, newType string) string {
	if !strings.Contains(msg, ": ") {
		return fmt.Sprintf("%s: %s", newType, msg)
	}
	parts := strings.SplitN(msg, ": ", 2)
	if strings.Contains(parts[0], "(") {
		typeParts := strings.SplitN(parts[0], "(", 2)
		scope := "(" + typeParts[1] // includes the closing parenthesis
		return fmt.Sprintf("%s%s: %s", newType, scope, parts[1])
	}
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

	// Create the commit.
	if err := g.Commit(opts.Message, opts.AllowEmpty, true); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}
	res := &CommitResult{ActualMessage: opts.Message}

	// Optionally push the commit.
	if opts.PushAfterCommit {
		branch, err := g.CurrentBranch()
		if err != nil {
			return nil, err
		}
		if err := g.Push(branch, false); err != nil {
			return nil, err
		}
		res.Pushed = true
	}

	return res, nil
}
