package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// SyncResult represents the outcome of a sync operation
type SyncResult struct {
	Success      bool
	NeedsAction  bool
	Action       string
	Message      string
	Conflicts    []string
	StashedFiles bool
	StashRef     string    // Reference to created stash
	OriginalRef  string    // Original HEAD before sync
	StartTime    time.Time // When sync started
}

// SyncBranch synchronizes the current branch with its parent (default) branch.
// It handles all common scenarios automatically and provides clear guidance when manual intervention is needed.
func SyncBranch(g git.Service, abort, cont bool) error {
	// Create a progress spinner
	spinner := ui.NewSpinner()
	defer spinner.Stop()

	// Verify repository state
	spinner.Start("Checking repository state")
	if err := verifyRepoState(g); err != nil {
		spinner.StopFail()
		return err
	}
	spinner.StopSuccess()

	// Handle abort/continue flags
	if result := handleSyncFlags(g, abort, cont); result.NeedsAction {
		return handleSyncResult(result)
	}

	// Start the sync process
	return performSync(g, spinner)
}

func verifyRepoState(g git.Service) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repository")
	}
	return nil
}

func handleSyncFlags(g git.Service, abort, cont bool) SyncResult {
	if abort {
		return handleAbort(g)
	}
	if cont {
		return handleContinue(g)
	}
	return SyncResult{Success: true}
}

func handleAbort(g git.Service) SyncResult {
	if merging, _ := g.IsMerging(); merging {
		if err := g.MergeAbort(); err != nil {
			return SyncResult{
				Success: false,
				Message: fmt.Sprintf("Failed to abort merge: %v", err),
			}
		}
		// Record the abort operation
		if err := RecordOperation(g, "merge", "Aborted merge", "git merge --abort", "merge", nil, "", "", false, ""); err != nil {
			ui.Warning("Failed to record merge abort in undo history")
		}
		return SyncResult{
			Success: true,
			Message: "Successfully aborted merge",
		}
	}
	if rebase, _ := g.IsRebasing(); rebase {
		if err := g.RebaseAbort(); err != nil {
			return SyncResult{
				Success: false,
				Message: fmt.Sprintf("Failed to abort rebase: %v", err),
			}
		}
		// Record the abort operation
		if err := RecordOperation(g, "rebase", "Aborted rebase", "git rebase --abort", "rebase", nil, "", "", false, ""); err != nil {
			ui.Warning("Failed to record rebase abort in undo history")
		}
		return SyncResult{
			Success: true,
			Message: "Successfully aborted rebase",
		}
	}
	return SyncResult{
		Success: false,
		Message: "No merge or rebase in progress to abort",
	}
}

func handleContinue(g git.Service) SyncResult {
	sg, ok := g.(*git.ShellGit)
	if !ok {
		return SyncResult{
			Success: false,
			Message: "Internal error: invalid git service type",
		}
	}

	if merging, _ := g.IsMerging(); merging {
		out, err := sg.MergeContinue()
		if err != nil {
			conflicts, _ := sg.ListConflictedFiles()
			return SyncResult{
				Success:     false,
				NeedsAction: true,
				Action:      "resolve_conflicts",
				Message:     "Merge conflicts need to be resolved",
				Conflicts:   strings.Split(conflicts, "\n"),
			}
		}
		// Record the continue operation
		if err := RecordOperation(g, "merge", "Continued merge", "git merge --continue", "merge", nil, "", "", false, ""); err != nil {
			ui.Warning("Failed to record merge continue in undo history")
		}
		return SyncResult{
			Success: true,
			Message: "Successfully continued merge: " + strings.TrimSpace(out),
		}
	}

	if rebase, _ := g.IsRebasing(); rebase {
		out, err := sg.RebaseContinue()
		if err != nil {
			conflicts, _ := sg.ListConflictedFiles()
			return SyncResult{
				Success:     false,
				NeedsAction: true,
				Action:      "resolve_conflicts",
				Message:     "Rebase conflicts need to be resolved",
				Conflicts:   strings.Split(conflicts, "\n"),
			}
		}
		// Record the continue operation
		if err := RecordOperation(g, "rebase", "Continued rebase", "git rebase --continue", "rebase", nil, "", "", false, ""); err != nil {
			ui.Warning("Failed to record rebase continue in undo history")
		}
		return SyncResult{
			Success: true,
			Message: "Successfully continued rebase: " + strings.TrimSpace(out),
		}
	}

	return SyncResult{
		Success: false,
		Message: "No merge or rebase in progress to continue",
	}
}

func performSync(g git.Service, spinner *ui.Spinner) error {
	var result SyncResult
	result.StartTime = time.Now()

	// 1. Get branch information
	spinner.Start("Getting branch information")
	curBranch, parentBranch, err := getBranchInfo(g)
	if err != nil {
		spinner.StopFail()
		return err
	}
	spinner.StopSuccess()

	// Save original ref for backup
	origRef, err := g.GetCommitHash("HEAD")
	if err != nil {
		ui.Warning("Could not save original ref for backup")
	}
	result.OriginalRef = origRef

	// 2. Stash changes if needed
	spinner.Start("Checking working directory")
	stashed, stashRef, err := handleWorkingDirectory(g)
	if err != nil {
		spinner.StopFail()
		return err
	}
	result.StashedFiles = stashed
	result.StashRef = stashRef
	spinner.StopSuccess()

	// Record stash operation if changes were stashed
	if stashed {
		if err := RecordOperation(g, "stash", "Stashed changes", "git stash", "stash", nil, curBranch, "", true, stashRef); err != nil {
			ui.Warning("Failed to record stash operation in undo history")
		}
	}

	// 3. Update parent branch
	spinner.Start(fmt.Sprintf("Updating %s", parentBranch))
	if err := updateParentBranch(g, parentBranch); err != nil {
		spinner.StopFail()
		return handleSyncError(g, err, &result)
	}
	spinner.StopSuccess()

	// 4. Rebase current branch
	spinner.Start(fmt.Sprintf("Rebasing %s onto %s", curBranch, parentBranch))
	if err := rebaseBranch(g, parentBranch); err != nil {
		spinner.StopFail()
		return handleSyncError(g, err, &result)
	}
	spinner.StopSuccess()

	// Record rebase operation
	if err := RecordOperation(g, "rebase", fmt.Sprintf("Rebased %s onto %s", curBranch, parentBranch), "git rebase", "rebase", nil, curBranch, "", stashed, stashRef); err != nil {
		ui.Warning("Failed to record rebase operation in undo history")
	}

	// Ensure we're on the correct branch after rebase
	if finalBranch, err := g.CurrentBranch(); err == nil && finalBranch != curBranch {
		if err := g.Checkout(curBranch); err != nil {
			ui.Warning(fmt.Sprintf("Failed to switch back to %s after rebase", curBranch))
		}
	}

	// 5. Pop stash if we stashed changes
	if result.StashedFiles {
		spinner.Start("Restoring stashed changes")
		if err := g.StashPop(); err != nil {
			spinner.StopFail()
			return handleSyncError(g, err, &result)
		}
		spinner.StopSuccess()

		// Record stash pop operation
		if err := RecordOperation(g, "stash", "Restored stashed changes", "git stash pop", "stash", nil, curBranch, "", false, ""); err != nil {
			ui.Warning("Failed to record stash pop operation in undo history")
		}
	}

	return nil
}

// SyncError represents an error that occurred during sync
type SyncError struct {
	Type      string
	Message   string
	Conflicts []string
}

func (e *SyncError) Error() string {
	switch e.Type {
	case "conflict":
		return fmt.Sprintf("%s in files:\n%s\n\nTo resolve:\n1. Fix conflicts in the files above\n2. Run 'git add' for each resolved file\n3. Run 'sage sync --continue'\n\nOr run 'sage sync --abort' to cancel",
			e.Message, strings.Join(e.Conflicts, "\n"))
	default:
		return e.Message
	}
}

func handleSyncResult(result SyncResult) error {
	if !result.Success {
		if result.NeedsAction {
			switch result.Action {
			case "resolve_conflicts":
				return &SyncError{
					Type:      "conflict",
					Message:   result.Message,
					Conflicts: result.Conflicts,
				}
			}
		}
		return fmt.Errorf(result.Message)
	}

	ui.Success(result.Message)
	return nil
}

func handleSyncError(g git.Service, err error, result *SyncResult) error {
	if strings.Contains(err.Error(), "conflict") {
		sg, ok := g.(*git.ShellGit)
		if !ok {
			return err
		}
		conflicts, _ := sg.ListConflictedFiles()
		return &SyncError{
			Type:      "conflict",
			Message:   "Conflicts detected during sync",
			Conflicts: strings.Split(conflicts, "\n"),
		}
	}
	return err
}

func getBranchInfo(g git.Service) (string, string, error) {
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current branch: %w", err)
	}

	parentBranch, err := g.DefaultBranch()
	if err != nil {
		return "", "", fmt.Errorf("failed to get default branch: %w", err)
	}

	return curBranch, parentBranch, nil
}

func handleWorkingDirectory(g git.Service) (bool, string, error) {
	clean, err := g.IsClean()
	if err != nil {
		return false, "", fmt.Errorf("failed to check working directory: %w", err)
	}

	if clean {
		return false, "", nil
	}

	// Stash changes with a descriptive message
	msg := fmt.Sprintf("sage-sync-%d", time.Now().Unix())
	if err := g.Stash(msg); err != nil {
		return false, "", fmt.Errorf("failed to stash changes: %w", err)
	}

	return true, msg, nil
}

func updateParentBranch(g git.Service, parentBranch string) error {
	// Get current branch before switching
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Switch to parent branch
	if err := g.Checkout(parentBranch); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", parentBranch, err)
	}

	// Update parent branch
	if err := g.PullFF(); err != nil {
		// Switch back to original branch before returning error
		_ = g.Checkout(curBranch)
		return fmt.Errorf("failed to update %s: %w", parentBranch, err)
	}

	// Switch back to original branch
	if err := g.Checkout(curBranch); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", curBranch, err)
	}

	return nil
}

func rebaseBranch(g git.Service, parentBranch string) error {
	// Get current branch
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Make sure we're on our feature branch
	if err := g.Checkout(curBranch); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", curBranch, err)
	}

	// Rebase current branch onto parent branch using the full command
	if err := g.RunInteractive("rebase", "--onto", parentBranch, parentBranch, curBranch); err != nil {
		return fmt.Errorf("failed to rebase onto %s: %w", parentBranch, err)
	}

	return nil
}
