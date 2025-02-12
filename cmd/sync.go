package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	syncAbort    bool
	syncContinue bool
	syncForce    bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current branch with default branch",
	Long: `Synchronize your current branch with the default branch (usually main or master).

This command will:
1. Automatically stash any uncommitted changes
2. Update the default branch with latest changes
3. Rebase your current branch on top of the default branch
4. Restore your stashed changes
5. Push the updated branch

If conflicts occur, it will:
1. Show which files have conflicts
2. Provide clear instructions on how to resolve them
3. Allow you to continue or abort the sync

Examples:
  sage sync              # Sync current branch with default branch
  sage sync --continue   # Continue a sync after resolving conflicts
  sage sync --abort      # Abort the current sync operation
  sage sync --force      # Force push after syncing (use with caution)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		// Show a warning if force flag is used
		if syncForce {
			ui.Warning("Using --force will overwrite remote branch history.")
			if !ui.Confirm("Are you sure you want to continue?") {
				return nil
			}
		}

		// Run the sync operation
		if err := app.SyncBranch(g, syncAbort, syncContinue); err != nil {
			// If it's a sync error with conflicts, show a more helpful message
			if syncErr, ok := err.(*app.SyncError); ok {
				ui.Error(syncErr.Error())
				return nil
			}
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&syncAbort, "abort", "a", false, "Abort merge/rebase")
	syncCmd.Flags().BoolVarP(&syncContinue, "continue", "c", false, "Continue merge/rebase")
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Force push after sync (use with caution)")

	// Make flags mutually exclusive
	syncCmd.MarkFlagsMutuallyExclusive("abort", "continue", "force")
}
