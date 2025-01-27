package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
)

// commitCmd represents "sage commit [message]"
var commitCmd = &cobra.Command{
	Use:           "commit [message]",
	Short:         "Stage all changes and commit with the provided message (or open an editor)",
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate arguments first
		if len(args) > 1 {
			return fmt.Errorf("accepts at most 1 arg(s)")
		}

		// Check if we're in a git repository
		if err := gitutils.RunGitCommand("rev-parse", "--git-dir"); err != nil {
			return fmt.Errorf("not a git repository")
		}

		var commitMsg string
		if len(args) == 1 {
			commitMsg = args[0]
		}

		// Stage all changes by default
		if err := gitutils.RunGitCommand("add", "."); err != nil {
			return err
		}

		if commitMsg == "" {
			// Open interactive editor if message not provided
			// By default, `git commit` will open the configured editor
			if err := gitutils.RunGitCommand("commit"); err != nil {
				return err
			}
		} else {
			// If user provided a message, use it
			if err := gitutils.RunGitCommand("commit", "-m", commitMsg); err != nil {
				return err
			}
		}

		fmt.Println("Commit successful.")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(commitCmd)
	// Additional flags: --amend, --no-verify, etc., can be added here
	// e.g. commitCmd.Flags().Bool("amend", false, "Amend the last commit")
	// and handle that logic in RunE if needed.
}
