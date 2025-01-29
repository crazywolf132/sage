package cmd

import (
	"fmt"
	"strings"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/spf13/cobra"
)

var (
	abortSync    bool
	continueSync bool
)

// syncCmd represents "sage sync"
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync current branch with parent branch",
	Long: `Sync will:
1. Check if the working tree is clean
2. Switch to parent branch and update it
3. Return to feature branch and merge parent
4. Handle any merge conflicts`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if abortSync && continueSync {
			return fmt.Errorf("cannot specify both --abort and --continue")
		}

		// Get current branch name
		currentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// Handle sync continue
		if continueSync {
			return handleSyncContinue(currentBranch)
		}

		// Handle sync abort
		if abortSync {
			return handleSyncAbort(currentBranch)
		}

		// Check if working tree is clean
		isClean, err := gitutils.IsWorkingTreeClean()
		if err != nil {
			return fmt.Errorf("failed to check working tree status: %w", err)
		}
		if !isClean {
			return fmt.Errorf("working tree is not clean. Please commit or stash your changes first")
		}

		// Get parent branch (usually main or master)
		parentBranch, err := getParentBranch()
		if err != nil {
			return fmt.Errorf("failed to determine parent branch: %w", err)
		}

		// Switch to parent branch
		fmt.Printf("Switching to %s...\n", parentBranch)
		if err := gitutils.DefaultRunner.RunGitCommand("checkout", parentBranch); err != nil {
			return fmt.Errorf("failed to switch to parent branch: %w", err)
		}

		// Update parent branch
		fmt.Println("Fetching latest changes...")
		if err := gitutils.DefaultRunner.RunGitCommand("fetch", "--all", "--prune"); err != nil {
			return fmt.Errorf("failed to fetch: %w", err)
		}

		fmt.Printf("Pulling latest changes for %s...\n", parentBranch)
		if err := gitutils.DefaultRunner.RunGitCommand("pull"); err != nil {
			return fmt.Errorf("failed to pull: %w", err)
		}

		// Switch back to feature branch
		fmt.Printf("Switching back to %s...\n", currentBranch)
		if err := gitutils.DefaultRunner.RunGitCommand("checkout", currentBranch); err != nil {
			return fmt.Errorf("failed to switch back to feature branch: %w", err)
		}

		// Merge parent branch
		fmt.Printf("Merging %s into %s...\n", parentBranch, currentBranch)
		err = gitutils.DefaultRunner.RunGitCommand("merge", parentBranch)
		if err != nil {
			// Check if there are merge conflicts
			conflicts, conflictErr := getMergeConflicts()
			if conflictErr != nil {
				return fmt.Errorf("failed to check for merge conflicts: %w", conflictErr)
			}

			if len(conflicts) > 0 {
				fmt.Println("\nMerge conflicts detected in the following files:")
				for _, conflict := range conflicts {
					fmt.Printf("  - %s\n", conflict)
				}
				fmt.Println("\nResolve the conflicts and run 'sage sync -c' to continue")
				fmt.Println("Or run 'sage sync -a' to abort the merge")
				return nil
			}
			return fmt.Errorf("merge failed: %w", err)
		}

		fmt.Printf("Successfully synced %s with %s\n", currentBranch, parentBranch)
		return nil
	},
}

func handleSyncContinue(currentBranch string) error {
	// Check if we're in a merge state
	inMerge, err := isInMerge()
	if err != nil {
		return fmt.Errorf("failed to check merge status: %w", err)
	}
	if !inMerge {
		return fmt.Errorf("no merge in progress")
	}

	// Get parent branch name from MERGE_HEAD
	parentBranch, err := gitutils.DefaultRunner.RunGitCommandWithOutput("name-rev", "--name-only", "MERGE_HEAD")
	if err != nil {
		return fmt.Errorf("failed to get parent branch name: %w", err)
	}
	// Clean up the branch name (remove remote prefix if present)
	parentBranch = strings.TrimPrefix(strings.TrimSpace(parentBranch), "remotes/origin/")

	// Create our custom merge message
	commitMsg := fmt.Sprintf("merge(%s): merged %s updates", parentBranch, parentBranch)

	// Complete the merge with our custom message
	if err := gitutils.DefaultRunner.RunGitCommand("commit", "-m", commitMsg); err != nil {
		return fmt.Errorf("failed to complete merge: %w", err)
	}

	fmt.Printf("Successfully completed merge on %s\n", currentBranch)
	return nil
}

func handleSyncAbort(currentBranch string) error {
	// Check if we're in a merge state
	inMerge, err := isInMerge()
	if err != nil {
		return fmt.Errorf("failed to check merge status: %w", err)
	}
	if !inMerge {
		return fmt.Errorf("no merge in progress")
	}

	// Abort the merge
	if err := gitutils.DefaultRunner.RunGitCommand("merge", "--abort"); err != nil {
		return fmt.Errorf("failed to abort merge: %w", err)
	}

	fmt.Printf("Successfully aborted merge on %s\n", currentBranch)
	return nil
}

func getParentBranch() (string, error) {
	// Try to get the configured parent branch
	output, err := gitutils.DefaultRunner.RunGitCommandWithOutput("config", "--get", "sage.parent-branch")
	if err == nil && output != "" {
		return strings.TrimSpace(output), nil
	}

	// Check if main exists
	if err := gitutils.DefaultRunner.RunGitCommand("rev-parse", "--verify", "main"); err == nil {
		return "main", nil
	}

	// Check if master exists
	if err := gitutils.DefaultRunner.RunGitCommand("rev-parse", "--verify", "master"); err == nil {
		return "master", nil
	}

	return "", fmt.Errorf("could not determine parent branch. Please configure using: git config sage.parent-branch <branch-name>")
}

func getMergeConflicts() ([]string, error) {
	output, err := gitutils.DefaultRunner.RunGitCommandWithOutput("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}

	conflicts := strings.Split(strings.TrimSpace(output), "\n")
	if len(conflicts) == 1 && conflicts[0] == "" {
		return []string{}, nil
	}
	return conflicts, nil
}

func isInMerge() (bool, error) {
	err := gitutils.DefaultRunner.RunGitCommand("rev-parse", "--verify", "MERGE_HEAD")
	if err != nil {
		// If MERGE_HEAD doesn't exist, we're not in a merge
		return false, nil
	}
	return true, nil
}

func init() {
	RootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&abortSync, "abort", "a", false, "Abort the current merge")
	syncCmd.Flags().BoolVarP(&continueSync, "continue", "c", false, "Continue the merge after resolving conflicts")
}
