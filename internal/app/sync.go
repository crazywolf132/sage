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
1. Run 'sage resolve' for interactive conflict resolution
2. Or resolve conflicts manually
3. Run 'sage sync --continue'

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
1. Run 'sage resolve' for interactive conflict resolution
2. Or resolve conflicts manually
3. Run 'sage sync --continue'

To start over: 'sage sync --abort'`)
	case "merge":
		return fmt.Sprintf(`Unable to automatically update your branch.

To continue:
1. Run 'sage resolve' for interactive conflict resolution
2. Or resolve conflicts manually
3. Run 'sage sync --continue'

To start over: 'sage sync --abort'`)
	default:
		return e.Message
	}
}

// SyncBranch synchronizes the current branch with its parent (default) branch.
// It handles all common scenarios automatically and provides clear guidance when manual intervention is needed.
func SyncBranch(g git.Service, opts SyncOptions) error {
	// Create a progress tracker
	progress := ui.NewSyncProgress()
	defer progress.ShowOptimizationTip()

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

	return performSync(g, opts, progress)
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
		err := sg.MergeContinue()
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
			Message: "Successfully continued merge",
		}
	}

	if rebase, _ := g.IsRebasing(); rebase {
		err := sg.RebaseContinue()
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
			Message: "Successfully continued rebase",
		}
	}

	return SyncResult{
		Success: false,
		Message: "No merge or rebase in progress to continue",
	}
}

func performSync(g git.Service, opts SyncOptions, progress *ui.SyncProgress) error {
	var result SyncResult
	result.StartTime = time.Now()

	if opts.DryRun {
		ui.Info("Dry run: Previewing sync operations without modifying your repository")
	}

	// 1. Repository Check
	progress.StartStep("verify")
	if err := verifyRepoState(g); err != nil {
		progress.CompleteStep("verify", false)
		return fmt.Errorf("Error: Not a Git repository. Please navigate to a valid Git project")
	}
	progress.CompleteStep("verify", true)

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
		progress.StartStep("stash")
		stashed, stashRef, err := handleWorkingDirectory(g)
		if err != nil {
			progress.CompleteStep("stash", false)
			return fmt.Errorf("Failed to stash changes: %w", err)
		}
		result.StashedFiles = stashed
		result.StashRef = stashRef
		progress.CompleteStep("stash", true)
	} else {
		progress.SkipStep("stash")
	}

	// 4. Remote Updates
	progress.StartStep("fetch")
	if err := g.FetchAll(); err != nil {
		progress.CompleteStep("fetch", false)
		if result.StashedFiles {
			restoreChanges(g, progress)
		}
		return fmt.Errorf("Failed to fetch updates: %w", err)
	}
	progress.CompleteStep("fetch", true)

	// Automatically pull changes
	progress.StartStep("pull")
	if err := g.Pull(); err != nil {
		progress.CompleteStep("pull", false)
		if result.StashedFiles {
			restoreChanges(g, progress)
		}
		return fmt.Errorf("Failed to pull updates: %w", err)
	}
	progress.CompleteStep("pull", true)

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

	// If we've diverged, choose strategy based on config or divergence
	if hasDiverged {
		progress.StartStep("integrate")
		// Check if user has specified a preferred strategy in config
		preferredStrategy := getPreferredMergeStrategy(g)
		divergence, err := g.GetBranchDivergence(curBranch, parentBranch)
		if err != nil {
			divergence = 0
		}

		if opts.Verbose {
			ui.Info(fmt.Sprintf("Branch has diverged by %d commits", divergence))
		}

		// Use preferred strategy if set, otherwise decide based on divergence
		switch preferredStrategy {
		case "merge":
			if opts.Verbose {
				ui.Info("Using merge strategy based on configuration")
			}
			if err := g.Merge(parentBranch); err != nil {
				progress.CompleteStep("integrate", false)
				if result.StashedFiles {
					restoreChanges(g, progress)
				}
				return fmt.Errorf("failed to merge %s: %w", parentBranch, err)
			}
		case "rebase":
			if opts.Verbose {
				ui.Info("Using rebase strategy based on configuration")
			}
			if err := rebaseBranch(g, parentBranch); err != nil {
				progress.CompleteStep("integrate", false)
				if result.StashedFiles {
					restoreChanges(g, progress)
				}
				return err
			}
		default:
			// Auto-select based on divergence
			if divergence > 10 {
				if opts.Verbose {
					ui.Info("Using merge strategy to preserve branch history")
				}
				ui.Info("Branch has diverged significantly - using merge strategy")
				if err := g.PullMerge(); err != nil {
					progress.CompleteStep("integrate", false)
					if result.StashedFiles {
						restoreChanges(g, progress)
					}
					return fmt.Errorf("failed to merge %s: %w", parentBranch, err)
				}
			} else {
				if opts.Verbose {
					ui.Info("Using rebase strategy for a clean history")
				}
				if err := rebaseBranch(g, parentBranch); err != nil {
					progress.CompleteStep("integrate", false)
					if result.StashedFiles {
						restoreChanges(g, progress)
					}
					return err
				}
			}
		}
		progress.CompleteStep("integrate", true)
	} else {
		progress.SkipStep("integrate")
	}

	// Then check if we're behind remote (if it exists)
	behind, err := isBehindRemote(g, curBranch)
	if err == nil && behind {
		needsUpdate = true
	}

	if needsUpdate {
		if !isMainBranch && !opts.NoPush {
			progress.StartStep("push")
			if err := pushChanges(g, curBranch, opts); err != nil {
				progress.CompleteStep("push", false)
				if result.StashedFiles {
					restoreChanges(g, progress)
				}
				return err
			}
			progress.CompleteStep("push", true)
		} else {
			progress.SkipStep("push")
		}
	} else {
		progress.SkipStep("push")
		if isMainBranch {
			ui.Info("Branch is up to date")
		}
	}

	// 7. Restore Changes
	if result.StashedFiles {
		return restoreChanges(g, progress)
	} else {
		progress.SkipStep("restore")
	}

	// 8. Final Status
	if isMainBranch {
		ui.Success("Branch is up to date")
	} else {
		ui.Success(fmt.Sprintf("Branch '%s' is now up to date", curBranch))
	}

	// Show a summary of what we did
	fmt.Println(progress.GetSummary())

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

	// If we've diverged, choose strategy based on config or divergence
	if hasDiverged {
		// Check if user has specified a preferred strategy in config
		preferredStrategy := getPreferredMergeStrategy(g)
		divergence, err := g.GetBranchDivergence(curBranch, parentBranch)
		if err != nil {
			divergence = 0
		}

		if opts.Verbose {
			ui.Info(fmt.Sprintf("Branch has diverged by %d commits", divergence))
		}

		// Use preferred strategy if set, otherwise decide based on divergence
		switch preferredStrategy {
		case "merge":
			if opts.Verbose {
				ui.Info("Using merge strategy based on configuration")
			}
			if err := g.Merge(parentBranch); err != nil {
				return fmt.Errorf("failed to merge %s: %w", parentBranch, err)
			}
		case "rebase":
			if opts.Verbose {
				ui.Info("Using rebase strategy based on configuration")
			}
			if err := rebaseBranch(g, parentBranch); err != nil {
				return err
			}
		default:
			// Auto-select based on divergence
			if divergence > 10 {
				if opts.Verbose {
					ui.Info("Using merge strategy to preserve branch history")
				}
				ui.Info("Branch has diverged significantly - using merge strategy")
				if err := g.PullMerge(); err != nil {
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
		}
	} else {
		// No divergence, we can fast-forward
		if err := g.Merge(parentBranch); err != nil {
			return fmt.Errorf("failed to fast-forward to %s: %w", parentBranch, err)
		}
	}

	return nil
}

// getPreferredMergeStrategy gets the user's preferred merge strategy from config
func getPreferredMergeStrategy(g git.Service) string {
	sg, ok := g.(*git.ShellGit)
	if !ok {
		return "" // default to auto-selection
	}

	// Try to get config from git or environment
	strategy, _ := sg.Run("config", "--get", "sage.merge.strategy")
	strategy = strings.TrimSpace(strategy)

	// Validate strategy
	switch strategy {
	case "merge", "rebase":
		return strategy
	default:
		return "" // auto-selection
	}
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
	// Get the merge base with remote
	remoteBranch := "origin/" + branch
	base, err := g.GetMergeBase(branch, remoteBranch)
	if err != nil {
		return false, err
	}

	// Get current HEAD
	head, err := g.GetCommitHash("HEAD")
	if err != nil {
		return false, err
	}

	// If merge base is different from HEAD, we're behind
	behind := base != head
	return behind, nil
}

// Helper to restore changes with progress tracking
func restoreChanges(g git.Service, progress *ui.SyncProgress) error {
	progress.StartStep("restore")
	if err := g.StashPop(); err != nil {
		progress.CompleteStep("restore", false)
		return &SyncError{
			Type:    "stash",
			Message: "Failed to restore your changes",
		}
	}
	progress.CompleteStep("restore", true)
	return nil
}
