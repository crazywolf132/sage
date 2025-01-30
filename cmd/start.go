package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crazywolf132/sage/internal/gitutils"
)

// hasUpstreamRemote checks if the repository has an upstream remote configured
func hasUpstreamRemote() (bool, error) {
	output, err := gitutils.RunGitCommandWithOutput("remote")
	if err != nil {
		return false, err
	}
	remotes := strings.Fields(output)
	for _, remote := range remotes {
		if remote == "upstream" {
			return true, nil
		}
	}
	return false, nil
}

// startCmd represents "sage start <branch-name>"
var startCmd = &cobra.Command{
	Use:   "start <branch-name>",
	Short: "Create and switch to a new branch from the default branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]

		// 1. Determine the default branch
		defaultBranch, err := gitutils.GetDefaultBranch()
		if err != nil {
			fmt.Println("Warning: Failed to determine default branch:", err)
			// Fallback to main if we can't determine default branch
			defaultBranch = "main"
		}

		// 2. Ensure working directory is clean or prompt user
		clean, err := gitutils.IsWorkingDirectoryClean()
		if err != nil {
			return err
		}
		if !clean {
			fmt.Println("\033[33mWARNING: You have uncommitted changes.\033[0m")
			// Optionally ask to stash or commit before proceeding.
		}

		// 3. Check for upstream remote
		hasUpstream, err := hasUpstreamRemote()
		if err != nil {
			return fmt.Errorf("failed to check for upstream remote: %w", err)
		}

		// 4. Fetch from all remotes
		fmt.Println("Fetching latest changes from remote(s)...")
		if err := gitutils.RunGitCommand("fetch", "--all", "--prune"); err != nil {
			return fmt.Errorf("failed to fetch from remotes: %w", err)
		}

		// 5. Checkout default branch
		if err := gitutils.RunGitCommand("switch", defaultBranch); err != nil {
			return fmt.Errorf("failed to switch to %s: %w", defaultBranch, err)
		}

		// 6. If this is a fork, sync with upstream first
		if hasUpstream {
			fmt.Printf("Fork detected. Syncing %s with upstream...\n", defaultBranch)
			// Fetch and merge from upstream
			if err := gitutils.RunGitCommand("fetch", "upstream"); err != nil {
				return fmt.Errorf("failed to fetch from upstream: %w", err)
			}
			if err := gitutils.RunGitCommand("merge", fmt.Sprintf("upstream/%s", defaultBranch)); err != nil {
				return fmt.Errorf("failed to merge upstream changes: %w", err)
			}
			// Push changes to origin to keep fork in sync
			fmt.Printf("Pushing synced changes to origin/%s...\n", defaultBranch)
			if err := gitutils.RunGitCommand("push", "origin", defaultBranch); err != nil {
				return fmt.Errorf("failed to push synced changes to origin: %w", err)
			}
		} else {
			// Regular pull from origin
			if err := gitutils.RunGitCommand("pull"); err != nil {
				return fmt.Errorf("failed to pull from origin: %w", err)
			}
		}

		// 7. Create new branch and switch
		if err := gitutils.RunGitCommand("switch", "-c", branchName); err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}
		fmt.Printf("Switched to a new branch '%s'\n", branchName)

		// 8. If --push is set, push branch to origin
		doPush, _ := cmd.Flags().GetBool("push")
		if doPush {
			if err := gitutils.RunGitCommand("push", "-u", "origin", branchName); err != nil {
				return fmt.Errorf("failed to push branch to origin: %w", err)
			}
			fmt.Printf("Branch '%s' pushed to origin\n", branchName)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().Bool("push", false, "Immediately push the new branch to the remote")
}
