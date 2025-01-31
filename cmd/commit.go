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
}
