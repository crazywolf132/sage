package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
)

// undoCmd represents "sage undo"
var undoCmd = &cobra.Command{
	Use:           "undo",
	Short:         "Undo the last Sage operation (revert commit, abort merge, etc.)",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if we're in a git repository
		if err := gitutils.RunGitCommand("rev-parse", "--git-dir"); err != nil {
			return fmt.Errorf("not a git repository")
		}

		// Check if there's an ongoing merge or rebase first
		inMerge, err := gitutils.IsMergeInProgress()
		if err != nil {
			return err
		}
		if inMerge {
			// Abort the merge
			if err := gitutils.RunGitCommand("merge", "--abort"); err != nil {
				return err
			}
			fmt.Println("Merge aborted successfully.")
			return nil
		}

		inRebase, err := gitutils.IsRebaseInProgress()
		if err != nil {
			return err
		}
		if inRebase {
			// Abort the rebase
			if err := gitutils.RunGitCommand("rebase", "--abort"); err != nil {
				return err
			}
			fmt.Println("Rebase aborted successfully.")
			return nil
		}

		// Otherwise, assume last operation was a commit; do a soft reset
		if err := gitutils.RunGitCommand("reset", "--soft", "HEAD~1"); err != nil {
			return err
		}
		fmt.Println("Last commit undone. Changes remain in the working directory.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(undoCmd)
}
