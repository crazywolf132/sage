package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	useConventional bool
	pushAfterCommit bool
)

// commitCmd represents "sage commit [message]"
var commitCmd = &cobra.Command{
	Use:           "commit [message]",
	Short:         "Stage all changes and commit with the provided message (or use interactive form)",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate arguments first
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s)")
		}

		// Check if we're in a git repository
		if err := gitutils.DefaultRunner.RunGitCommand("rev-parse", "--git-dir"); err != nil {
			return fmt.Errorf("not a git repository")
		}

		// Stage all changes by default
		if err := gitutils.DefaultRunner.RunGitCommand("add", "."); err != nil {
			return err
		}

		var commitMsg string
		if len(args) == 1 {
			commitMsg = args[0]
		} else {
			// Use interactive form when no message provided
			form, err := ui.GetCommitDetails(useConventional)
			if err != nil {
				return err
			}

			if useConventional {
				// Format conventional commit message
				if form.Scope != "" {
					commitMsg = fmt.Sprintf("%s(%s): %s", form.Type, form.Scope, form.Message)
				} else {
					commitMsg = fmt.Sprintf("%s: %s", form.Type, form.Message)
				}
			} else {
				commitMsg = form.Message
			}
		}

		// Create the commit with the message
		if err := gitutils.DefaultRunner.RunGitCommand("commit", "-m", commitMsg); err != nil {
			return err
		}

		fmt.Println("Commit successful.")

		// Push if requested
		if pushAfterCommit {
			currentBranch, err := gitutils.GetCurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}

			fmt.Printf("Pushing to origin/%s...\n", currentBranch)
			if err := gitutils.DefaultRunner.RunGitCommand("push", "origin", currentBranch); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}
			fmt.Println("Push successful.")
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVarP(&useConventional, "conventional", "c", false, "Use conventional commit format")
	commitCmd.Flags().BoolVarP(&pushAfterCommit, "push", "p", false, "Push changes after committing")
}
