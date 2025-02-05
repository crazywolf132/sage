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
		cur, err := g.CurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		db, err := g.DefaultBranch()
		if err != nil {
			db = "main"
		}

		var continueErr error
		if merging {
			continueErr = g.RunInteractive("merge", "--continue")
		} else if rebase {
			continueErr = g.RunInteractive("rebase", "--continue")
		} else {
			return fmt.Errorf("no merge or rebase in progress to continue")
		}

		if continueErr != nil {
			return continueErr
		}

		// After successful continue, create conventional commit and push
		if merging {
			msg := fmt.Sprintf("merge(%s): sync with %s", cur, db)
			if err := g.Commit(msg, false); err != nil {
				return fmt.Errorf("failed to create merge commit: %w", err)
			}
		}

		// Push the changes
		if err := g.Push(cur, false); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}

		fmt.Printf("%s Successfully continued and pushed changes to %s\n", ui.Green("✓"), cur)
		return nil
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
	if err := g.PullRebase(); err != nil {
		// If pull fails, it might be because:
		// 1. The branch doesn't exist on remote
		// 2. There are conflicts during rebase
		if strings.Contains(err.Error(), "no tracking information") {
			fmt.Printf("%s Branch %q not found on remote, skipping update\n", ui.Sage("ℹ"), cur)
		} else if strings.Contains(err.Error(), "error: could not apply") {
			return fmt.Errorf("rebase conflicts in %s - fix & run 'git rebase --continue' or 'sage sync -a' to abort", cur)
		} else {
			return fmt.Errorf("failed to update current branch: %w", err)
		}
	}

	// Now sync with default branch
	fmt.Printf("%s Syncing with %q\n", ui.Sage("ℹ"), db)
	if err := g.Checkout(db); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", db, err)
	}

	if err := g.PullRebase(); err != nil {
		if strings.Contains(err.Error(), "error: could not apply") {
			return fmt.Errorf("rebase conflicts in %s - fix & run 'git rebase --continue' or 'sage sync -a' to abort", db)
		}
		return fmt.Errorf("failed to update %s: %w", db, err)
	}

	if err := g.Checkout(cur); err != nil {
		return fmt.Errorf("failed to checkout back to %s: %w", cur, err)
	}

	// do a rebase instead of merge
	if err := g.RunInteractive("rebase", db); err != nil {
		if strings.Contains(err.Error(), "error: could not apply") {
			return fmt.Errorf("rebase conflicts - fix & run 'git rebase --continue' or 'sage sync -a' to abort")
		}
		return fmt.Errorf("failed to rebase %s onto %s: %w", cur, db, err)
	}

	fmt.Printf("%s Successfully rebased %s onto %s\n", ui.Green("✓"), cur, db)
	return nil
}
