package app

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/ai"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// CommitOptions defines the options for a commit.
type CommitOptions struct {
	Message         string
	AllowEmpty      bool
	PushAfterCommit bool
	UseConventional bool // if true, use our conventional commit prompt
	UseAI           bool // if true, generate commit message using AI if no message is provided
	ChangeType      string
	StageAll        bool // (legacy; not used in our new logic)
}

// CommitResult contains the result of the commit.
type CommitResult struct {
	ActualMessage string
	Pushed        bool
}

// changeCommitType changes the conventional commit type.
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

// Commit implements the simplified commit logic.
// It stages everything except files under ".sage/" (unless those are already staged)
// and then, if no commit message is provided, either generates one via AI (if UseAI is true)
// or prompts the user.
func Commit(g git.Service, opts CommitOptions) (*CommitResult, error) {
	// 1. Ensure we’re in a Git repo.
	isRepo, err := g.IsRepo()
	if err != nil || !isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	// 2. Get the current status.
	status, err := g.StatusPorcelain()
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	if strings.TrimSpace(status) == "" && !opts.AllowEmpty {
		return nil, fmt.Errorf("no changes to commit")
	}

	// 3. If no commit message was provided, generate one.
	if opts.Message == "" {
		if opts.UseAI {
			// Use AI to generate commit message.
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
			var choice string
			err = survey.AskOne(&survey.Select{
				Message: "Choose an option:",
				Options: []string{"Accept", "Enter manually"},
			}, &choice)
			if err != nil {
				return nil, err
			}
			if choice == "Accept" {
				opts.Message = aiMsg
			} else {
				// Fallback to manual prompt.
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
		} else {
			// Otherwise, ask the user for a commit message.
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

	// 4. If a commit type change is requested, update the message.
	if opts.ChangeType != "" {
		opts.Message = changeCommitType(opts.Message, opts.ChangeType)
	}

	// 5. Check whether any file under .sage/ is staged.
	sageStaged, err := g.IsPathStaged(".sage/")
	if err != nil {
		return nil, err
	}
	// If .sage/ is not staged, stage every file except those in ".sage/".
	if !sageStaged {
		if err := g.StageAllExcept([]string{".sage/"}); err != nil {
			return nil, err
		}
	}

	// 6. Create the commit.
	if err := g.Commit(opts.Message, opts.AllowEmpty, opts.StageAll); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}
	res := &CommitResult{ActualMessage: opts.Message}

	// 7. Optionally push the commit.
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
