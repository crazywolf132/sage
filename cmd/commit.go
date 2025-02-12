package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	commitMessage      string
	commitEmpty        bool
	commitPush         bool
	commitConventional bool
	commitAI           bool
	commitAutoAccept   bool
)

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Stage and commit changes",
	Long: `Stage and commit changes in one step.

Examples:
  # Interactive commit message prompt with AI support
  sage commit --ai

  # Direct commit with message
  sage commit "feat: add user authentication"

  # Stage everything (including .sage/) and commit with push
  sage commit -p "fix: resolve null pointer error"`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"c"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			commitMessage = args[0]
		}

		g := git.NewShellGit()

		res, err := app.Commit(g, app.CommitOptions{
			Message:         commitMessage,
			AllowEmpty:      commitEmpty,
			PushAfterCommit: commitPush,
			UseConventional: commitConventional,
			UseAI:           commitAI,
			AutoAccept:      commitAutoAccept,
		})
		if err != nil {
			return err
		}

		fmt.Printf("%s Created commit", ui.Green("✓"))
		if res.Pushed {
			fmt.Printf(" and pushed")
		}
		fmt.Printf(": %s\n", res.ActualMessage)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVar(&commitEmpty, "empty", false, "Allow empty commits")
	commitCmd.Flags().BoolVarP(&commitPush, "push", "p", false, "Push after commit")
	commitCmd.Flags().BoolVarP(&commitConventional, "conventional", "c", false, "Use conventional commit format")
	commitCmd.Flags().BoolVarP(&commitAI, "ai", "a", false, "Use AI to generate commit message")
	commitCmd.Flags().BoolVarP(&commitAutoAccept, "yes", "y", false, "Automatically accept AI-generated commit message")
}
