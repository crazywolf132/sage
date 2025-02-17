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
	syncNoPush   bool
	syncDryRun   bool
	syncVerbose  bool
	syncTarget   string
)

var syncCmd = &cobra.Command{
	Use:   "sync [target-branch]",
	Short: "Magically sync your branch with the latest changes",
	Long: `Sync keeps your branch up to date with the latest changes, handling all the complex Git operations for you.

What it does:
✨ Updates your branch with the latest changes
✨ Handles conflicts gracefully
✨ Keeps your work safe

Just run 'sage sync' and let the magic happen!

Common Scenarios:
  • Just sync:        sage sync
  • Sync with main:   sage sync main
  • Fix conflicts:    sage sync --continue
  • Start over:       sage sync --abort

Advanced Options:
  • Preview changes:  sage sync --dry-run
  • Skip auto-push:   sage sync --no-push
  • Show details:     sage sync --verbose`,
	Example: `  sage sync
  sage sync main
  sage sync --continue
  sage sync --abort`,
	Aliases: []string{"update", "s"},
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		// Get target branch from args or use default
		if len(args) > 0 {
			syncTarget = args[0]
		}

		// Validate flags
		if syncAbort && syncContinue {
			return ui.NewError("Hint: Use either --abort to start over or --continue after fixing conflicts")
		}

		// Run the sync operation with all options
		opts := app.SyncOptions{
			TargetBranch: syncTarget,
			NoPush:       syncNoPush,
			DryRun:       syncDryRun,
			Verbose:      syncVerbose,
			Abort:        syncAbort,
			Continue:     syncContinue,
		}

		if err := app.SyncBranch(g, opts); err != nil {
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
