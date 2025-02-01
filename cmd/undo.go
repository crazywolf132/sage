package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo the last commit or abort merge",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		return app.Undo(g)
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
}
