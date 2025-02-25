package app

import (
	"fmt"
	"strconv"
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

	// Log settings only at top level to avoid duplicates
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

	// Check if we need to update
	needsUpdate := false

	// Get current and parent HEADs
	currentHead, err := g.GetCommitHash(curBranch)
	if err != nil {
		return fmt.Errorf("failed to get current HEAD: %w", err)
	}

	// Get the merge base (common ancestor)
	mergeBase, err := g.GetMergeBase(curBranch, parentBranch)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Determine if we have diverged (have unique commits)
	hasDiverged := mergeBase != currentHead

	// If we've diverged significantly (more than 10 commits), use merge
	// Otherwise, use rebase for a cleaner history
	if hasDiverged {
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
	} else {
		// No divergence, we can fast-forward
		if err := g.Merge(parentBranch); err != nil {
			return fmt.Errorf("failed to fast-forward to %s: %w", parentBranch, err)
		}
	}

	// Then check if we're behind remote (if it exists)
	behind, err := isBehindRemote(g, curBranch)
	if opts.Verbose {
		if err != nil {
			ui.Info(fmt.Sprintf("Remote branch check: Error=%v", err))
		} else {
			ui.Info(fmt.Sprintf("Remote branch check: Behind=%v", behind))
		}
	}
	if err == nil && behind {
		// Pull remote changes for the current branch before integrating with parent
		// Only do this if the branch has a remote tracking branch
		spinner.Start(fmt.Sprintf("Pulling changes from remote for branch '%s'...", curBranch))
		// Try to pull, and handle error if no upstream branch
		if err := g.PullFF(); err != nil {
			// We already checked that the branch has a remote tracking branch,
			// so any error here is a real error
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
		needsUpdate = true
	}

	if needsUpdate {
		// 5. Integration (only if not on main/master)
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
		if !opts.NoPush && !isMainBranch {
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

	// Get the merge base (common ancestor)
	mergeBase, err := g.GetMergeBase(curBranch, parentBranch)
	if err != nil {
		return fmt.Errorf("failed to get merge base: %w", err)
	}

	// Get current HEAD
	currentHead, err := g.GetCommitHash(curBranch)
	if err != nil {
		return fmt.Errorf("failed to get current HEAD: %w", err)
	}

	// Determine if we have diverged (have unique commits)
	hasDiverged := mergeBase != currentHead

	// If we've diverged significantly (more than 10 commits), use merge
	// Otherwise, use rebase for a cleaner history
	if hasDiverged {
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
	} else {
		// No divergence, we can fast-forward
		if err := g.Merge(parentBranch); err != nil {
			return fmt.Errorf("failed to fast-forward to %s: %w", parentBranch, err)
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
		// The underlying git methods now handle upstream setup automatically
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
	// First, get the actual upstream branch name
	out, err := g.Run("rev-parse", "--abbrev-ref", branch+"@{upstream}")

	// If there's no upstream branch, we're not behind
	if err != nil {
		if strings.Contains(err.Error(), "no upstream") ||
			strings.Contains(err.Error(), "no tracking information") {
			return false, fmt.Errorf("no upstream branch configured")
		}
		return false, fmt.Errorf("upstream check failed: %w", err)
	}

	remoteBranch := strings.TrimSpace(out)
	fmt.Printf("DEBUG: Using remote branch: %s for local branch: %s\n", remoteBranch, branch)

	// Get remote branch hash
	remoteHash, err := g.GetCommitHash(remoteBranch)
	if err != nil {
		return false, fmt.Errorf("remote branch check failed: %w", err)
	}

	// Get local branch hash for debugging
	localHash, err := g.GetCommitHash(branch)
	if err != nil {
		return false, fmt.Errorf("local branch check failed: %w", err)
	}

	// Debug: Print the hashes
	fmt.Printf("DEBUG: Local hash: %s, Remote hash: %s\n", localHash, remoteHash)

	// Check if we're ahead, behind, or diverged from remote
	revListOut, err := g.Run("rev-list", "--left-right", "--count", branch+"..."+remoteBranch)
	if err != nil {
		return false, fmt.Errorf("rev-list failed: %w", err)
	}

	// Debug: Print the raw output
	fmt.Printf("DEBUG: rev-list output: '%s'\n", revListOut)

	// Parse output like "1	2" where first number is ahead count, second is behind count
	parts := strings.Fields(revListOut)
	if len(parts) != 2 {
		return false, fmt.Errorf("unexpected output format from git rev-list: %s", revListOut)
	}

	// Parse the counts
	aheadCount, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to parse ahead count: %w", err)
	}

	behindCount, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse behind count: %w", err)
	}

	// Debug: Print the counts
	fmt.Printf("DEBUG: Ahead count: %d, Behind count: %d\n", aheadCount, behindCount)

	// We're behind if the behind count is greater than 0
	return behindCount > 0, nil
}
