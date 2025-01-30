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
	allowEmpty      bool
)

// commitCmd represents "sage commit [message]"
var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Stage and commit changes",
	Long: `Stage all changes and create a commit. Optionally use conventional commit format.
	
To trigger GitHub Actions without changes:
1. Use --empty to create an empty commit (leaves commit history)
2. Consider using 'workflow_dispatch' in your GitHub Actions for a cleaner solution
   See: https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#workflow_dispatch`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if we're in a git repository
		if err := gitutils.RunGitCommand("rev-parse", "--git-dir"); err != nil {
			return fmt.Errorf("not a git repository")
		}

		// Check if there are any changes to commit (skip if empty commit is requested)
		if !allowEmpty {
			clean, err := gitutils.IsWorkingDirectoryClean()
			if err != nil {
				return err
			}
			if clean {
				return fmt.Errorf("no changes to commit (use --empty to create an empty commit)")
			}
		}

		// Get commit message
		var commitForm ui.CommitForm
		if len(args) > 0 {
			// Use provided message
			commitForm.Message = args[0]
		} else {
			// Get message through interactive form
			var err error
			commitForm, err = ui.GetCommitDetails(useConventional)
			if err != nil {
				return fmt.Errorf("failed to get commit details: %w", err)
			}
		}

		// Build commit message
		var commitMsg string
		if useConventional {
			if commitForm.Scope != "" {
				commitMsg = fmt.Sprintf("%s(%s): %s", commitForm.Type, commitForm.Scope, commitForm.Message)
			} else {
				commitMsg = fmt.Sprintf("%s: %s", commitForm.Type, commitForm.Message)
			}
		} else {
			commitMsg = commitForm.Message
		}

		fmt.Println("\n🔄 Preparing commit...")

		// Stage changes if not empty commit
		if !allowEmpty {
			fmt.Println("   📝 Staging changes")
			if err := gitutils.RunGitCommand("add", "."); err != nil {
				return fmt.Errorf("failed to stage changes: %w", err)
			}
		}

		// Create commit
		fmt.Println("   ✨ Creating commit")
		args = []string{"commit", "-m", commitMsg}
		if allowEmpty {
			args = append(args, "--allow-empty")
			fmt.Println("   ℹ️  Creating empty commit for CI trigger")
		}

		if err := gitutils.RunGitCommand(args...); err != nil {
			return fmt.Errorf("failed to create commit: %w", err)
		}

		fmt.Printf("\n✨ Changes committed!\n")
		fmt.Printf("   %s\n", commitMsg)

		// Push if requested
		if pushAfterCommit {
			fmt.Println("\n🔄 Publishing changes...")
			if err := gitutils.RunGitCommand("push"); err != nil {
				return fmt.Errorf("failed to push changes: %w", err)
			}
			fmt.Println("   ⬆️  Changes published to remote")
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVarP(&useConventional, "conventional", "c", false, "Use conventional commit format")
	commitCmd.Flags().BoolVarP(&pushAfterCommit, "push", "p", false, "Push changes after committing")
	commitCmd.Flags().BoolVar(&allowEmpty, "empty", false, "Allow empty commits (useful for triggering CI)")
}
