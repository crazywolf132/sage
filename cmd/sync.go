package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"
)

var (
	syncAbort    bool
	syncContinue bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current branch with default branch",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		return app.SyncBranch(g, syncAbort, syncContinue)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&syncAbort, "abort", "a", false, "Abort merge/rebase")
	syncCmd.Flags().BoolVarP(&syncContinue, "continue", "c", false, "Continue merge/rebase")
}
