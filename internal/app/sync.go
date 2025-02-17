package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// SyncOptions contains all options for the sync operation
type SyncOptions struct {
	TargetBranch string
	NoPush       bool
	DryRun       bool
	Verbose      bool
	Abort        bool
	Continue     bool
}

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

// SyncError represents an error that occurred during sync
type SyncError struct {
	Type      string
	Message   string
	Conflicts []string
}

func (e *SyncError) Error() string {
	switch e.Type {
	case "conflict":
		return fmt.Sprintf(`Conflicts found in these files:
%s

To resolve:
1. Open the files
2. Resolve conflicts
3. Save changes
4. Run 'sage sync --continue'

To start over: 'sage sync --abort'`,
			strings.Join(e.Conflicts, "\n"))
	case "diverged":
		return fmt.Sprintf(`Remote branch has new changes.

%s

To update:
1. Use 'sage sync --force' (recommended)
2. Or merge manually and run 'sage sync'`, e.Message)
	case "stash":
		return fmt.Sprintf(`%s

Your changes are safely stashed.
Run 'git stash pop' to restore them.`, e.Message)
	case "rebase":
		return fmt.Sprintf(`Unable to automatically update your branch.

To continue:
1. Resolve any conflicts
2. Run 'sage sync --continue'

To start over: 'sage sync --abort'`)
	case "merge":
		return fmt.Sprintf(`Unable to automatically update your branch.

To continue:
1. Resolve any conflicts
2. Run 'sage sync --continue'

To start over: 'sage sync --abort'`)
	default:
		return e.Message
	}
}

// SyncBranch synchronizes the current branch with its parent (default) branch.
// It handles all common scenarios automatically and provides clear guidance when manual intervention is needed.
func SyncBranch(g git.Service, opts SyncOptions) error {
	spinner := ui.NewSpinner()
	defer spinner.Stop()

	if opts.DryRun {
		ui.Info("Dry run: Previewing sync operations without modifying your repository")
	}
	if opts.Verbose {
		ui.Info("Verbose mode: Displaying detailed operation logs")
	}

	// Handle abort/continue flags first
	if result := handleSyncFlags(g, opts.Abort, opts.Continue); result.NeedsAction {
		return handleSyncResult(result)
	}

	return performSync(g, opts, spinner)
}

func verifyRepoState(g git.Service) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repository")
	}

	// Check if we're in a detached HEAD state
	head, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}
	if head == "HEAD" {
		return fmt.Errorf("cannot sync in detached HEAD state")
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

func performSync(g git.Service, opts SyncOptions, spinner *ui.Spinner) error {
	var result SyncResult
	result.StartTime = time.Now()

	if opts.DryRun {
		ui.Info("Dry run: Previewing sync operations without modifying your repository")
	}

	// 1. Repository Check
	spinner.Start("Verifying repository...")
	if err := verifyRepoState(g); err != nil {
		spinner.StopFail()
		return fmt.Errorf("Error: Not a Git repository. Please navigate to a valid Git project")
	}
	spinner.StopSuccess()

	// 2. Branch Information
	curBranch, parentBranch, err := getBranchInfo(g, opts.TargetBranch)
	if err != nil {
		return err
	}

	// Save reference for safety
	origRef, _ := g.GetCommitHash("HEAD")
	result.OriginalRef = origRef

	// 3. Local Changes Check
	hasChanges, err := hasUncommittedChanges(g)
	if err != nil {
		return err
	}
	if hasChanges {
		spinner.Start("Saving work in progress...")
		stashed, stashRef, err := handleWorkingDirectory(g)
		if err != nil {
			spinner.StopFail()
			return fmt.Errorf("Failed to stash changes: %w", err)
		}
		result.StashedFiles = stashed
		result.StashRef = stashRef
		spinner.StopSuccess()
	}

	// 4. Remote Updates
	spinner.Start("Fetching updates...")
	if err := g.FetchAll(); err != nil {
		spinner.StopFail()
		if result.StashedFiles {
			restoreSpinner := ui.NewSpinner()
			restoreSpinner.Start("Restoring your work...")
			_ = g.StashPop()
			restoreSpinner.StopSuccess()
		}
		return fmt.Errorf("Failed to fetch updates: %w", err)
	}
	spinner.StopSuccess()

	// Check if we're on main/master branch
	isMainBranch := curBranch == parentBranch
	behind, err := isBehindRemote(g, curBranch)
	if err == nil && behind {
		// 5. Integration (only if not on main/master or if behind remote)
		if !isMainBranch {
			spinner.Start("Integrating remote changes...")
			if err := integrateChanges(g, parentBranch, opts); err != nil {
				spinner.StopFail()
				if result.StashedFiles {
					restoreSpinner := ui.NewSpinner()
					restoreSpinner.Start("Restoring your work...")
					_ = g.StashPop()
					restoreSpinner.StopSuccess()
				}
				return handleSyncError(g, err, &result)
			}
			spinner.StopSuccess()
		}

		// 6. Push Changes (only if not on main/master or if we have local commits)
		if !opts.NoPush && (!isMainBranch || behind) {
			spinner.Start("Pushing changes...")
			if err := pushChanges(g, curBranch, opts); err != nil {
				spinner.StopFail()
				if result.StashedFiles {
					restoreSpinner := ui.NewSpinner()
					restoreSpinner.Start("Restoring your work...")
					_ = g.StashPop()
					restoreSpinner.StopSuccess()
				}
				return err
			}
			spinner.StopSuccess()
		}
	} else if isMainBranch {
		ui.Info("Branch is up to date")
	}

	// 7. Restore Changes
	if result.StashedFiles {
		spinner.Start("Restoring your work...")
		if err := g.StashPop(); err != nil {
			spinner.StopFail()
			return &SyncError{
				Type:    "stash",
				Message: "Failed to restore your changes",
			}
		}
		spinner.StopSuccess()
	}

	// 8. Final Status
	if isMainBranch {
		ui.Success("Branch is up to date")
	} else {
		ui.Success(fmt.Sprintf("Branch '%s' is now up to date", curBranch))
	}

	return nil
}

func getBranchInfo(g git.Service, targetBranch string) (string, string, error) {
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return "", "", fmt.Errorf("failed to get current branch: %w", err)
	}

	parentBranch := targetBranch
	if parentBranch == "" {
		parentBranch, err = g.DefaultBranch()
		if err != nil {
			return "", "", fmt.Errorf("failed to get default branch: %w", err)
		}
	}

	// Validate target branch exists
	branches, err := g.ListBranches()
	if err != nil {
		return "", "", fmt.Errorf("failed to list branches: %w", err)
	}
	branchExists := false
	for _, b := range branches {
		if b == parentBranch {
			branchExists = true
			break
		}
	}
	if !branchExists {
		return "", "", fmt.Errorf("target branch '%s' does not exist", parentBranch)
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

func hasUncommittedChanges(g git.Service) (bool, error) {
	clean, err := g.IsClean()
	if err != nil {
		return false, fmt.Errorf("failed to check working directory: %w", err)
	}
	return !clean, nil
}

func integrateChanges(g git.Service, parentBranch string, opts SyncOptions) error {
	curBranch, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Determine the best integration strategy
	divergence, err := g.GetBranchDivergence(curBranch, parentBranch)
	if err != nil {
		divergence = 0
	}

	if divergence > 10 {
		if opts.Verbose {
			ui.Info("Using merge strategy to preserve branch history")
		}
		if err := g.Merge(parentBranch); err != nil {
			return fmt.Errorf("failed to merge %s: %w", parentBranch, err)
		}
	} else {
		if opts.Verbose {
			ui.Info("Using rebase strategy for a clean history")
		}
		if err := rebaseBranch(g, parentBranch); err != nil {
			return err
		}
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

	// Rebase current branch onto parent branch
	if err := g.RunInteractive("rebase", "--onto", parentBranch, parentBranch, curBranch); err != nil {
		return fmt.Errorf("failed to rebase onto %s: %w", parentBranch, err)
	}

	return nil
}

func pushChanges(g git.Service, branch string, opts SyncOptions) error {
	// Always use --force-with-lease for safety
	if err := g.PushWithLease(branch); err != nil {
		if strings.Contains(err.Error(), "no upstream branch") {
			// Try to set up the upstream branch
			if err := g.RunInteractive("push", "--set-upstream", "origin", branch); err != nil {
				return fmt.Errorf("failed to set upstream branch: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to push changes: %w", err)
	}
	return nil
}

func handleSyncResult(result SyncResult) error {
	if !result.Success {
		if result.NeedsAction {
			switch result.Action {
			case "resolve_conflicts":
				return &SyncError{
					Type:      "conflict",
					Message:   "Merge conflicts need to be resolved",
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
			Message:   "Merge conflicts detected. Please resolve conflicts and run 'sage sync --continue'",
			Conflicts: strings.Split(conflicts, "\n"),
		}
	}

	if strings.Contains(err.Error(), "failed to rebase") || strings.Contains(err.Error(), "failed to merge") {
		return &SyncError{
			Type:    "conflict",
			Message: "Merge conflicts detected. Please resolve conflicts and run 'sage sync --continue'",
		}
	}

	return err
}

func isBehindRemote(g git.Service, branch string) (bool, error) {
	// Get the merge base with remote
	base, err := g.GetMergeBase(branch, "origin/"+branch)
	if err != nil {
		return false, err
	}

	// Get current HEAD
	head, err := g.GetCommitHash("HEAD")
	if err != nil {
		return false, err
	}

	// If merge base is different from HEAD, we're behind
	return base != head, nil
}
