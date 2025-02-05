package app

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

type CommitOptions struct {
	Message         string
	UseConventional bool
	AllowEmpty      bool
	PushAfterCommit bool
	UseAI           bool
	AutoAcceptAI    bool
	SuggestType     string // Type to suggest to AI
	ChangeType      string // Type to change to without regenerating
}

type CommitResult struct {
	ActualMessage string
	Pushed        bool
}

// changeCommitType changes the type of a conventional commit message
func changeCommitType(msg, newType string) string {
	if !strings.Contains(msg, ": ") {
		return fmt.Sprintf("%s: %s", newType, msg)
	}

	parts := strings.SplitN(msg, ": ", 2)
	oldType := parts[0]

	// Handle scoped commits (e.g., feat(api): message)
	if strings.Contains(oldType, "(") {
		typeParts := strings.SplitN(oldType, "(", 2)
		scope := "(" + typeParts[1] // includes the closing parenthesis
		return fmt.Sprintf("%s%s: %s", newType, scope, parts[1])
	}

	return fmt.Sprintf("%s: %s", newType, parts[1])
}

func Commit(g git.Service, opts CommitOptions) (*CommitResult, error) {
	isRepo, err := g.IsRepo()
	if err != nil {
		return nil, err
	}
	if !isRepo {
		return nil, fmt.Errorf("no a git repo")
	}

	if opts.UseAI && !opts.AllowEmpty {
		// Stage all files first
		if err := g.StageAll(); err != nil {
			return nil, fmt.Errorf("failed to stage files: %w", err)
		}

		diff, err := g.GetDiff()
		if err != nil {
			return nil, fmt.Errorf("failed to get diff for AI commit message: %w", err)
		}
		if diff == "" {
			return nil, fmt.Errorf("no changes to commit; use --empty to allow empty")
		}
		client := ai.NewClient("")
		for {
			var msg string
			var err error

			if opts.SuggestType != "" {
				msg, err = client.GenerateCommitMessage(diff + "\n\nPlease use the commit type: " + opts.SuggestType)
			} else {
				msg, err = client.GenerateCommitMessage(diff)
			}

			if err != nil {
				return nil, fmt.Errorf("failed to generate AI commit message: %w", err)
			}

			// Ensure the message is in conventional commit format
			if !strings.Contains(msg, ":") {
				msg = "chore: " + msg
			}

			if opts.AutoAcceptAI {
				opts.Message = msg
				break
			}

			fmt.Printf("Generated commit message: %q\n", msg)
			confirm := ""
			err = survey.AskOne(&survey.Select{
				Message: "What would you like to do?",
				Options: []string{
					"Accept",
					"Regenerate",
					"Change type",
					"Enter manually",
				},
			}, &confirm)
			if err != nil {
				return nil, err
			}

			switch confirm {
			case "Accept":
				opts.Message = msg
				break
			case "Change type":
				newType := ""
				err = survey.AskOne(&survey.Select{
					Message: "Select new commit type:",
					Options: []string{
						"feat", "fix", "docs", "style",
						"refactor", "test", "chore",
					},
				}, &newType)
				if err != nil {
					return nil, err
				}
				opts.Message = changeCommitType(msg, newType)
				break
			case "Enter manually":
				opts.UseAI = false
				opts.Message = ""
				break
				// For "Regenerate", continue the loop
			}

			if opts.Message != "" || !opts.UseAI {
				break
			}
		}
	} else if opts.Message == "" {
		// prompt
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
	} else if opts.UseConventional && !strings.Contains(opts.Message, ":") {
		opts.Message = "chore: " + opts.Message
	}

	// Handle type change if requested
	if opts.ChangeType != "" {
		opts.Message = changeCommitType(opts.Message, opts.ChangeType)
	}

	if !opts.AllowEmpty {
		clean, err := g.IsClean()
		if err != nil {
			return nil, err
		}
		if clean {
			return nil, fmt.Errorf("no changes to commit; use --empty to allow empty")
		}
		if err := g.StageAll(); err != nil {
			return nil, err
		}
	}

	if err := g.Commit(opts.Message, opts.AllowEmpty); err != nil {
		return nil, err
	}
	res := &CommitResult{ActualMessage: opts.Message}

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
