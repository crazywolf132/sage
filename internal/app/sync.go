package app

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

func SyncBranch(g git.Service, abort, cont bool) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repo")
	}

	merging, err := g.IsMerging()
	if err != nil {
		return err
	}

	rebase, err := g.IsRebasing()
	if err != nil {
		return err
	}

	if abort {
		if merging {
			return g.MergeAbort()
		}
		if rebase {
			return g.RebaseAbort()
		}
		return fmt.Errorf("no merge or rebase to abort")
	}
	if cont {
		// not fully implemented
		return fmt.Errorf("merges or rebase continue not yet implemented here")
	}

	// Get branch names
	db, err := g.DefaultBranch()
	if err != nil {
		db = "main"
	}
	cur, err := g.CurrentBranch()
	if err != nil {
		return err
	}

	fmt.Printf("%s Fetching all remotes\n", ui.Sage("ℹ"))
	if err := g.FetchAll(); err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	// First update the current branch from remote if it exists there
	clean, err := g.IsClean()
	if err != nil {
		return fmt.Errorf("failed to check working directory: %w", err)
	}
	if !clean {
		return fmt.Errorf("working directory not clean, please commit or stash changes first")
	}

	fmt.Printf("%s Updating current branch %q\n", ui.Sage("ℹ"), cur)
	if err := g.Pull(); err != nil {
		// If pull fails, it might be because the branch doesn't exist on remote yet
		if !strings.Contains(err.Error(), "no tracking information") {
			return fmt.Errorf("failed to update current branch: %w", err)
		}
		fmt.Printf("%s Branch %q not found on remote, skipping update\n", ui.Sage("ℹ"), cur)
	}

	// Now sync with default branch
	fmt.Printf("%s Syncing with %q\n", ui.Sage("ℹ"), db)
	if err := g.Checkout(db); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", db, err)
	}

	if err := g.Pull(); err != nil {
		return fmt.Errorf("failed to update %s: %w", db, err)
	}

	if err := g.Checkout(cur); err != nil {
		return fmt.Errorf("failed to checkout back to %s: %w", cur, err)
	}

	// do a merge
	if err := g.Merge(db); err != nil {
		if strings.Contains(err.Error(), "CONFLICT") {
			return fmt.Errorf("conflict - fix & run 'sage sync -c' or 'sage sync -a'")
		}
		return fmt.Errorf("failed to merge %s into %s: %w", db, cur, err)
	}

	fmt.Printf("%s Successfully synced %s with %s\n", ui.Green("✓"), cur, db)
	return nil
}
