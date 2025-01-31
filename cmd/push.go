package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"

	"github.com/AlecAivazis/survey/v2"
)

var (
	forcePush bool
	skipYes   bool
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push changes to remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		if forcePush && !skipYes {
			fmt.Println(ui.Red("WARNING: You're about to force-push."))
			var confirm bool
			if err := survey.AskOne(&survey.Confirm{Message: "Are you sure?"}, &confirm); err != nil {
				return err
			}
			if !confirm {
				fmt.Println(ui.Gray("Cancelled."))
				return nil
			}
		}
		err := app.PushCurrentBranch(g, forcePush)
		if err != nil {
			return err
		}
		if forcePush {
			fmt.Println(ui.Green("Force-pushed current branch."))
		} else {
			fmt.Println(ui.Green("Pushed current branch."))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVarP(&forcePush, "force", "f", false, "Force push")
	pushCmd.Flags().BoolVarP(&skipYes, "yes", "y", false, "Skip confirmation for force push")
}
