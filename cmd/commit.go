package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	commitMessage    string
	commitEmpty      bool
	commitPush       bool
	commitAI         bool
	commitAutoAccept bool
	commitType       string
	commitNewType    string
	commitMultiple   bool
)

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Stage and commit changes",
	Long: `Stage and commit changes in one step. Without a message, prompts for one.
	
Examples:
  # Interactive commit message prompt
  sage commit

  # Direct commit with message
  sage commit "feat: add user authentication"

  # Use AI to generate commit message
  sage commit --ai

  # Multiple commits with AI grouping
  sage commit --multiple --ai

  # Auto-accept AI suggestions
  sage commit --ai --yes`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			commitMessage = args[0]
		}

		g := git.NewShellGit()

		if commitMultiple {
			if !commitAI {
				return fmt.Errorf("--multiple requires --ai flag")
			}
			return app.CommitMultiple(g, app.CommitOptions{
				Message:         commitMessage,
				AllowEmpty:      commitEmpty,
				PushAfterCommit: commitPush,
				AutoAcceptAI:    commitAutoAccept,
				SuggestType:     commitType,
				ChangeType:      commitNewType,
			})
		}

		res, err := app.Commit(g, app.CommitOptions{
			Message:         commitMessage,
			AllowEmpty:      commitEmpty,
			PushAfterCommit: commitPush,
			UseAI:           commitAI,
			AutoAcceptAI:    commitAutoAccept,
			SuggestType:     commitType,
			ChangeType:      commitNewType,
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
	commitCmd.Flags().BoolVarP(&commitAI, "ai", "a", false, "Use AI to generate commit message")
	commitCmd.Flags().BoolVarP(&commitMultiple, "multiple", "m", false, "Create multiple commits based on AI grouping (requires --ai)")
	commitCmd.Flags().BoolVarP(&commitAutoAccept, "yes", "y", false, "Auto-accept AI suggestions")
	commitCmd.Flags().StringVar(&commitType, "type", "", "Suggest this commit type to AI")
	commitCmd.Flags().StringVar(&commitNewType, "change-type", "", "Change commit type without regenerating")
}
