package cmd

import (
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/spf13/cobra"
)

var (
	syncAbort    bool
	syncContinue bool
	syncNoPush   bool
	syncDryRun   bool
	syncVerbose  bool
	syncTarget   string
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize your branch with the main branch",
	Long: `Synchronize your branch with updates from the parent branch.

The 'sync' command intelligently pulls changes from the parent branch
and ensures your work stays up to date. It automatically:

1. Saves any work in progress (staged and unstaged changes)
2. Updates your branch with new changes from parent branch
3. Restores your work
4. Resolves conflicts when possible

By default, it uses the main branch as the parent branch, but you
can specify any branch with the --target flag.`,
	Example: `  # Sync with the main branch (default)
  sage sync

  # Sync with a specific branch
  sage sync --target develop

  # Sync without pushing changes
  sage sync --no-push

  # Resume after resolving conflicts
  sage sync --continue

  # Abort a conflicted sync
  sage sync --abort`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		// Run sync with options
		opts := app.SyncOptions{
			TargetBranch: syncTarget,
			NoPush:       syncNoPush,
			DryRun:       syncDryRun,
			Verbose:      syncVerbose,
			Abort:        syncAbort,
			Continue:     syncContinue,
		}

		if err := app.SyncBranch(g, opts); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Common flags
	syncCmd.Flags().BoolVarP(&syncContinue, "continue", "c", false, "Continue after fixing conflicts")
	syncCmd.Flags().BoolVarP(&syncAbort, "abort", "a", false, "Start over if something goes wrong")

	// Advanced flags
	syncCmd.Flags().BoolVar(&syncNoPush, "no-push", false, "Skip pushing changes to remote")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview sync operations without making changes")
	syncCmd.Flags().BoolVar(&syncVerbose, "verbose", false, "Show detailed operation logs")

	// Make certain flags mutually exclusive
	syncCmd.MarkFlagsMutuallyExclusive("abort", "continue")
	syncCmd.MarkFlagsMutuallyExclusive("dry-run", "continue")
	syncCmd.MarkFlagsMutuallyExclusive("dry-run", "abort")
}
