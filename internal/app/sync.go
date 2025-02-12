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
	var backupBranch string

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

	// 2. Stash changes if needed
	spinner.Start("Checking working directory")
	stashed, stashRef, err := handleWorkingDirectory(g)
	if err != nil {
		spinner.StopFail()
		return err
	}
	spinner.StopSuccess()

	// 3. Create backup branch if we have the original ref
	if origRef != "" {
		backupBranch = fmt.Sprintf("%s-backup-%s", curBranch, time.Now().Format("20060102-150405"))
		if err := g.CreateBranch(backupBranch); err != nil {
			ui.Warning(fmt.Sprintf("Could not create backup branch %s", backupBranch))
			backupBranch = ""
		} else {
			ui.Info(fmt.Sprintf("Created backup branch: %s", backupBranch))
		}
	}

	// Defer cleanup of backup branch
	defer func() {
		if backupBranch != "" {
			// Only clean up if sync was successful (no error returned)
			if err == nil {
				spinner.Start(fmt.Sprintf("Cleaning up backup branch %s", backupBranch))
				if err := g.DeleteBranch(backupBranch); err != nil {
					spinner.StopFail()
					ui.Warning(fmt.Sprintf("Could not delete backup branch %s: %v", backupBranch, err))
				} else {
					spinner.StopSuccess()
					ui.Info(fmt.Sprintf("Deleted backup branch: %s", backupBranch))
				}
			} else {
				ui.Info(fmt.Sprintf("Keeping backup branch for recovery: %s", backupBranch))
			}
		}
	}()

	// 4. Fetch remote changes
	spinner.Start("Fetching remote changes")
	if err := g.FetchAll(); err != nil {
		spinner.StopFail()
		return fmt.Errorf("failed to fetch remotes: %w", err)
	}
	spinner.StopSuccess()

	// 5. Check remote status
	spinner.Start("Checking remote status")
	if diverged, err := hasRemoteDiverged(g, curBranch); err != nil {
		ui.Warning("Could not check remote branch status")
	} else if diverged {
		spinner.StopFail()
		return fmt.Errorf("remote branch has diverged. Use 'git pull' first or use --force to overwrite")
	}
	spinner.StopSuccess()

	// 6. Handle sync based on branch type
	if curBranch == parentBranch {
		err = syncDefaultBranch(g, spinner, parentBranch, stashed, stashRef)
	} else {
		err = syncFeatureBranch(g, spinner, curBranch, parentBranch, stashed, stashRef)
	}

	return err
}

func getBranchInfo(g git.Service) (current, parent string, err error) {
	current, err = g.CurrentBranch()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current branch: %w", err)
	}

	parent, err = g.DefaultBranch()
	if err != nil || parent == "" {
		parent = "main"
	}

	return current, parent, nil
}

func handleWorkingDirectory(g git.Service) (stashed bool, stashRef string, err error) {
	clean, err := g.IsClean()
	if err != nil {
		return false, "", fmt.Errorf("failed to check working directory: %w", err)
	}

	if !clean {
		// Generate unique stash reference
		stashRef = fmt.Sprintf("sage-sync-%s", time.Now().Format("20060102-150405"))

		// Create specific stash for this sync
		if err := g.Stash(stashRef); err != nil {
			return false, "", fmt.Errorf("failed to stash changes: %w", err)
		}

		// Verify stash was created
		stashes, err := g.StashList()
		if err != nil || len(stashes) == 0 {
			return false, "", fmt.Errorf("failed to verify stash creation: %w", err)
		}

		return true, stashRef, nil
	}

	return false, "", nil
}

func hasRemoteDiverged(g git.Service, branch string) (bool, error) {
	// Get local and remote refs
	localRef, err := g.GetCommitHash("HEAD")
	if err != nil {
		return false, err
	}

	remoteRef, err := g.GetCommitHash("origin/" + branch)
	if err != nil {
		// If remote ref doesn't exist, it hasn't diverged
		if strings.Contains(err.Error(), "no such ref") {
			return false, nil
		}
		return false, err
	}

	// Check if remote contains commits we don't have
	contained, err := g.IsAncestor(localRef, remoteRef)
	if err != nil {
		return false, err
	}

	return !contained, nil
}

func syncDefaultBranch(g git.Service, spinner *ui.Spinner, branch string, stashed bool, stashRef string) error {
	spinner.Start(fmt.Sprintf("Updating %s branch", branch))
	if err := g.PullRebase(); err != nil {
		spinner.StopFail()
		return fmt.Errorf("failed to update branch %q: %w", branch, err)
	}
	spinner.StopSuccess()

	if stashed {
		return restoreStashedChanges(g, spinner, stashRef)
	}
	return nil
}

func syncFeatureBranch(g git.Service, spinner *ui.Spinner, current, parent string, stashed bool, stashRef string) error {
	// 1. Update parent branch
	spinner.Start(fmt.Sprintf("Updating %s branch", parent))
	if err := updateParentBranch(g, parent); err != nil {
		spinner.StopFail()
		return err
	}
	spinner.StopSuccess()

	// 2. Switch back and rebase
	spinner.Start(fmt.Sprintf("Rebasing %s onto %s", current, parent))
	if err := rebaseOntoBranch(g, current, parent); err != nil {
		spinner.StopFail()
		if syncErr, ok := err.(*SyncError); ok {
			return handleConflicts(g, syncErr.Conflicts)
		}
		return err
	}
	spinner.StopSuccess()

	// 3. Push changes
	spinner.Start("Pushing changes")
	if err := g.Push(current, false); err != nil {
		spinner.StopFail()
		return handlePushError(err, current)
	}
	spinner.StopSuccess()

	// 4. Restore stashed changes if any
	if stashed {
		return restoreStashedChanges(g, spinner, stashRef)
	}
	return nil
}

func updateParentBranch(g git.Service, parent string) error {
	if err := g.Checkout(parent); err != nil {
		return fmt.Errorf("failed to checkout %q: %w", parent, err)
	}

	// Use simple pull for default branch - it's safer and more predictable
	if err := g.Pull(); err != nil {
		return fmt.Errorf("failed to update %q: %w", parent, err)
	}
	return nil
}

func rebaseOntoBranch(g git.Service, current, parent string) error {
	if err := g.Checkout(current); err != nil {
		return fmt.Errorf("failed to checkout %q: %w", current, err)
	}

	// First, check if merge would be cleaner than rebase
	conflicts, err := g.GetBranchMergeConflicts(current)
	if err == nil && conflicts > 0 {
		// If we detect potential conflicts, try merge instead
		if err := g.Merge(parent); err != nil {
			sg, ok := g.(*git.ShellGit)
			if !ok {
				return fmt.Errorf("merge failed: %w", err)
			}

			conflicts, _ := sg.ListConflictedFiles()
			return &SyncError{
				Type:      "conflict",
				Message:   "Merge conflicts detected",
				Conflicts: strings.Split(conflicts, "\n"),
			}
		}
		return nil
	}

	// Check how diverged the branches are
	divergeCount, err := g.GetBranchDivergence(current, parent)
	if err == nil && divergeCount > 10 {
		// If branches have diverged significantly, prefer merge
		if err := g.Merge(parent); err != nil {
			sg, ok := g.(*git.ShellGit)
			if !ok {
				return fmt.Errorf("merge failed: %w", err)
			}

			conflicts, _ := sg.ListConflictedFiles()
			return &SyncError{
				Type:      "conflict",
				Message:   "Merge conflicts detected",
				Conflicts: strings.Split(conflicts, "\n"),
			}
		}
		return nil
	}

	// For simple cases or when merge analysis fails, try rebase
	if err := g.RunInteractive("rebase", parent); err != nil {
		sg, ok := g.(*git.ShellGit)
		if !ok {
			return fmt.Errorf("rebase failed: %w", err)
		}

		conflicts, _ := sg.ListConflictedFiles()
		return &SyncError{
			Type:      "conflict",
			Message:   "Rebase conflicts detected",
			Conflicts: strings.Split(conflicts, "\n"),
		}
	}
	return nil
}

func handlePushError(err error, branch string) error {
	if strings.Contains(strings.ToLower(err.Error()), "non-fast-forward") {
		return fmt.Errorf("remote branch has new changes. Please run 'sage sync' again to incorporate them")
	}
	return fmt.Errorf("failed to push branch %q: %w", branch, err)
}

func restoreStashedChanges(g git.Service, spinner *ui.Spinner, stashRef string) error {
	spinner.Start("Restoring your changes")
	if err := g.StashPop(); err != nil {
		spinner.StopFail()
		return fmt.Errorf("failed to restore your changes: %w\nYou can restore them manually with 'git stash pop'", err)
	}
	spinner.StopSuccess()
	return nil
}

func handleConflicts(g git.Service, conflicts []string) error {
	ui.Warning("Conflicts detected in the following files:")
	for _, file := range conflicts {
		fmt.Printf("  - %s\n", file)
	}

	ui.Info("\nTo resolve conflicts:")
	fmt.Println("1. Open each conflicted file and look for conflict markers:")
	fmt.Println("   <<<<<<< HEAD")
	fmt.Println("   Your changes")
	fmt.Println("   =======")
	fmt.Println("   Their changes")
	fmt.Println("   >>>>>>> branch-name")
	fmt.Println("\n2. Edit the files to resolve conflicts")
	fmt.Println("3. Remove the conflict markers")
	fmt.Println("4. Stage the resolved files:")
	fmt.Printf("   git add %s\n", strings.Join(conflicts, " "))
	fmt.Println("5. Continue the sync:")
	fmt.Println("   sage sync --continue")

	return &SyncError{
		Type:      "conflict",
		Message:   "Conflicts need to be resolved",
		Conflicts: conflicts,
	}
}

// SyncError represents a sync-specific error with additional context
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
