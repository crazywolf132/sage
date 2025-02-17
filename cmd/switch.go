package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"

	"strings"

	"github.com/AlecAivazis/survey/v2"
)

var switchCmd = &cobra.Command{
	Use:     "switch [branch]",
	Short:   "Switch to existing branch (or prompt)",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"sw", "checkout", "co"},
	Example: `  # Switch to a branch (with auto-completion)
  sage switch feature/awesome

  # Interactive branch selection
  sage switch

  # Switch using partial branch name
  sage switch feat<TAB>  # Will complete to feature/awesome`,
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
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		g := git.NewShellGit()
		branches, err := g.ListBranches()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		// Filter branches based on the partial input
		if toComplete != "" {
			filtered := make([]string, 0)
			for _, branch := range branches {
				if strings.HasPrefix(branch, toComplete) {
					filtered = append(filtered, branch)
				}
			}
			return filtered, cobra.ShellCompDirectiveNoFileComp
		}

		return branches, cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
