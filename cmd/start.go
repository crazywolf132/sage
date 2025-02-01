package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	startPush bool
)

var startCmd = &cobra.Command{
	Use:   "start <branch>",
	Short: "Create & switch to a new branch from default branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newBranch := args[0]
		g := git.NewShellGit()
		if err := app.StartBranch(g, newBranch, startPush); err != nil {
			return err
		}
		fmt.Printf("%s Created & switched to '%s'\n", ui.Green("✓"), newBranch)
		if startPush {
			fmt.Println("   Also pushed.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVar(&startPush, "push", false, "Push branch after creation")
}
