package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"

	"github.com/AlecAivazis/survey/v2"
)

var switchCmd = &cobra.Command{
	Use:   "switch [branch]",
	Short: "Switch to existing branch (or prompt)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		branches, err := g.ListBranches()
		if err != nil {
			return err
		}

		var target string
		if len(args) == 1 {
			target = args[0]
		} else {
			if len(branches) == 0 {
				return nil
			}
			err := survey.AskOne(&survey.Select{
				Message: "Pick a branch:",
				Options: branches,
			}, &target)
			if err != nil {
				return err
			}
		}
		return app.SwitchBranch(g, target)
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
