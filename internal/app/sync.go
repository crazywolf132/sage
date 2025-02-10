package app

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// SyncBranch synchronizes the current branch with its parent (default) branch.
// It supports three modes:
// 1. If abort is true, it aborts an in‑progress merge or rebase.
// 2. If cont (continue) is true, it automatically continues a merge or rebase using non‑interactive commands.
// 3. Otherwise, it performs a full automated sync:
//   - fetches remotes,
//   - updates the default branch,
//   - switches back to the current branch,
//   - rebases the current branch onto the default branch,
//   - and pushes the result.
//
// In case of rebase conflicts, it lists the conflicted files and instructs the user to resolve them manually.
func SyncBranch(g git.Service, abort, cont bool) error {
	// Verify that we are in a Git repository.
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repository")
	}

	// If the abort flag is provided, try to abort any in‑progress merge or rebase.
	if abort {
		if merging, _ := g.IsMerging(); merging {
			ui.Info("Aborting merge...")
			return g.MergeAbort()
		}
		if rebase, _ := g.IsRebasing(); rebase {
			ui.Info("Aborting rebase...")
			return g.RebaseAbort()
		}
		return fmt.Errorf("no merge or rebase in progress to abort")
	}

	// If the continue flag is provided, use non‑interactive commands to continue.
	if cont {
		// For non‑interactive execution, we need our concrete Git type.
		sg, ok := g.(*git.ShellGit)
		if !ok {
			return fmt.Errorf("git service is not of type *git.ShellGit")
		}
		if merging, _ := g.IsMerging(); merging {
			ui.Info("Continuing merge...")
			out, err := sg.MergeContinue()
			if err != nil {
				return err
			}
			ui.Info("Merge continued: " + strings.TrimSpace(out))
		} else if rebase, _ := g.IsRebasing(); rebase {
			ui.Info("Continuing rebase...")
			out, err := sg.RebaseContinue()
			if err != nil {
				// List conflicted files for a nicer message.
				conflicts, _ := sg.ListConflictedFiles()
				return fmt.Errorf("rebase conflicts encountered in files:\n%s\nPlease resolve conflicts and run 'sage sync --continue'", conflicts)
			}
			ui.Info("Rebase continued: " + strings.TrimSpace(out))
		} else {
			return fmt.Errorf("no merge or rebase in progress to continue")
		}
		// After continuing, push the current branch.
		cur, err := g.CurrentBranch()
		if err != nil {
			return err
		}
		if err := g.Push(cur, false); err != nil {
			return fmt.Errorf("failed to push branch %q: %w", cur, err)
		}
		ui.Info("Successfully continued and pushed changes.")
		return nil
	}

	// Otherwise, perform a full automated sync.
	// 1. Get current branch and default branch.
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	parentBranch, err := g.DefaultBranch()
	if err != nil || parentBranch == "" {
		parentBranch = "main"
	}

	// 2. Fetch all remote changes.
	ui.Info("Fetching remote changes...")
	if err := g.FetchAll(); err != nil {
		return fmt.Errorf("failed to fetch remotes: %w", err)
	}

	// 3. Ensure the working directory is clean.
	clean, err := g.IsClean()
	if err != nil {
		return fmt.Errorf("failed to check working directory: %w", err)
	}
	if !clean {
		return fmt.Errorf("working directory not clean; please commit or stash changes")
	}

	// 4. If the current branch is the default branch, simply update it.
	if curBranch == parentBranch {
		ui.Info("You are on the default branch. Pulling latest changes...")
		if err := g.PullRebase(); err != nil {
			return fmt.Errorf("failed to update branch %q: %w", parentBranch, err)
		}
		ui.Info(fmt.Sprintf("Branch %q updated successfully.", parentBranch))
		return nil
	}

	// 5. Update the parent branch.
	ui.Info(fmt.Sprintf("Checking out default branch %q...", parentBranch))
	if err := g.Checkout(parentBranch); err != nil {
		return fmt.Errorf("failed to checkout default branch %q: %w", parentBranch, err)
	}
	ui.Info(fmt.Sprintf("Pulling latest changes on %q...", parentBranch))
	if err := g.PullRebase(); err != nil {
		return fmt.Errorf("failed to update default branch %q: %w", parentBranch, err)
	}
	ui.Info(fmt.Sprintf("Default branch %q updated.", parentBranch))

	// 6. Switch back to the current branch.
	ui.Info(fmt.Sprintf("Switching back to branch %q...", curBranch))
	if err := g.Checkout(curBranch); err != nil {
		return fmt.Errorf("failed to checkout branch %q: %w", curBranch, err)
	}

	// 7. Rebase the current branch onto the updated default branch.
	ui.Info(fmt.Sprintf("Rebasing branch %q onto %q...", curBranch, parentBranch))
	if err := g.RunInteractive("rebase", parentBranch); err != nil {
		// If rebase fails, try to list conflicted files.
		sg, ok := g.(*git.ShellGit)
		var conflictMsg string
		if ok {
			conflicts, _ := sg.ListConflictedFiles()
			conflictMsg = conflicts
		} else {
			conflictMsg = "unknown conflicts"
		}
		return fmt.Errorf("rebase failed due to conflicts in files:\n%s\nPlease resolve conflicts manually and run 'sage sync --continue' or abort with 'sage sync --abort'", conflictMsg)
	}
	ui.Info("Rebase complete.")

	// 8. Push the current branch.
	ui.Info(fmt.Sprintf("Pushing branch %q...", curBranch))
	if err := g.Push(curBranch, false); err != nil {
		return fmt.Errorf("failed to push branch %q: %w", curBranch, err)
	}
	ui.Info(fmt.Sprintf("Branch %q successfully synced.", curBranch))
	return nil
}
