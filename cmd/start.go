package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	startNoPush bool
)

var startCmd = &cobra.Command{
	Use:   "start <branch>",
	Short: "Create & switch to a new branch from default branch",
	Long: `Create a new branch from the default branch (usually main), switch to it, and push it to remote.
	
Examples:
  # Create a new branch and push it
  sage start feature/awesome

  # Create a new branch without pushing
  sage start feature/local --no-push`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newBranch := args[0]
		g := git.NewShellGit()
		if err := app.StartBranch(g, newBranch, !startNoPush); err != nil {
			return err
		}
		fmt.Printf("%s Created & switched to '%s'\n", ui.Green("✓"), newBranch)
		if !startNoPush {
			fmt.Printf("%s Pushed branch to remote\n", ui.Green("✓"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVar(&startNoPush, "no-push", false, "Don't push branch after creation")
}
