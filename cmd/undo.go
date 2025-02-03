package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"
)

var (
	undoCount int
)

var undoCmd = &cobra.Command{
	Use:   "undo [count]",
	Short: "Undo the last commit(s) or abort merge",
	Long: `Undo the last commit(s) or abort an ongoing merge/rebase.
If a count is specified, undoes that many commits.
Examples:
  sage undo     # Undo last commit
  sage undo 3   # Undo last 3 commits`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		return app.Undo(g, undoCount)
	},
}

func init() {
	rootCmd.AddCommand(undoCmd)
	undoCmd.Flags().IntVarP(&undoCount, "count", "n", 1, "Number of commits to undo")
}
