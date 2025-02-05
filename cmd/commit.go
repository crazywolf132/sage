package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	commitMsg       string
	useConventional bool
	allowEmpty      bool
	pushAfterCommit bool
	useAI           bool
	autoAcceptAI    bool
	suggestType     string
	changeType      string
)

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Stage & commit changes",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			commitMsg = args[0]
		}
		g := git.NewShellGit()
		res, err := app.Commit(g, app.CommitOptions{
			Message:         commitMsg,
			UseConventional: useConventional,
			AllowEmpty:      allowEmpty,
			PushAfterCommit: pushAfterCommit,
			UseAI:           useAI,
			AutoAcceptAI:    autoAcceptAI,
			SuggestType:     suggestType,
			ChangeType:      changeType,
		})
		if err != nil {
			return err
		}

		fmt.Printf("%s Committed with message: %q\n", ui.Sage("✔"), res.ActualMessage)
		if res.Pushed {
			fmt.Println("   Pushed to remote.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringVarP(&commitMsg, "message", "m", "", "Commit message (if not specified, interactive prompt)")
	commitCmd.Flags().BoolVarP(&useConventional, "conventional", "c", false, "Use conventional commit style")
	commitCmd.Flags().BoolVar(&allowEmpty, "empty", false, "Allow empty commit")
	commitCmd.Flags().BoolVarP(&pushAfterCommit, "push", "p", false, "Push after committing")
	commitCmd.Flags().BoolVarP(&useAI, "ai", "a", false, "Use AI to generate commit message")
	commitCmd.Flags().BoolVarP(&autoAcceptAI, "yes", "y", false, "Automatically accept AI generated commit message")
	commitCmd.Flags().StringVar(&suggestType, "suggest-type", "", "Suggest a commit type to the AI (feat, fix, docs, etc)")
	commitCmd.Flags().StringVar(&changeType, "type", "", "Change the type of the generated commit without regenerating")
}
