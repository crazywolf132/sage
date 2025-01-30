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
			fmt.Println("⚠️  Could not determine default branch, using 'main'")
			defaultBranch = "main"
		}

		// 2. Ensure working directory is clean or prompt user
		clean, err := gitutils.IsWorkingDirectoryClean()
		if err != nil {
			return err
		}
		if !clean {
			fmt.Println("⚠️  You have uncommitted changes")
		}

		// 3. Check for upstream remote
		hasUpstream, err := hasUpstreamRemote()
		if err != nil {
			return fmt.Errorf("failed to check for upstream remote: %w", err)
		}

		fmt.Printf("\n🔄 Setting up branch '%s'...\n", branchName)

		// 4. Fetch from all remotes
		fmt.Println("   📡 Fetching latest changes")
		if err := gitutils.RunGitCommand("fetch", "--all", "--prune"); err != nil {
			return fmt.Errorf("failed to fetch from remotes: %w", err)
		}

		// 5. Checkout default branch
		fmt.Printf("   ⎇  Switching to %s\n", defaultBranch)
		if err := gitutils.RunGitCommand("switch", defaultBranch); err != nil {
			return fmt.Errorf("failed to switch to %s: %w", defaultBranch, err)
		}

		// 6. If this is a fork, sync with upstream first
		if hasUpstream {
			fmt.Printf("   🔄 Syncing with upstream/%s\n", defaultBranch)
			// Fetch and merge from upstream
			if err := gitutils.RunGitCommand("fetch", "upstream"); err != nil {
				return fmt.Errorf("failed to fetch from upstream: %w", err)
			}
			if err := gitutils.RunGitCommand("merge", fmt.Sprintf("upstream/%s", defaultBranch)); err != nil {
				return fmt.Errorf("failed to merge upstream changes: %w", err)
			}
			// Push changes to origin to keep fork in sync
			fmt.Printf("   ⬆️  Updating origin/%s\n", defaultBranch)
			if err := gitutils.RunGitCommand("push", "origin", defaultBranch); err != nil {
				return fmt.Errorf("failed to push synced changes to origin: %w", err)
			}
		} else {
			// Regular pull from origin
			fmt.Println("   ⬇️  Pulling latest changes")
			if err := gitutils.RunGitCommand("pull"); err != nil {
				return fmt.Errorf("failed to pull from origin: %w", err)
			}
		}

		// 7. Create new branch and switch
		fmt.Printf("   🌱 Creating new branch\n")
		if err := gitutils.RunGitCommand("switch", "-c", branchName); err != nil {
			return fmt.Errorf("failed to create branch %s: %w", branchName, err)
		}

		// 8. If --push is set, push branch to origin
		doPush, _ := cmd.Flags().GetBool("push")
		if doPush {
			fmt.Printf("   ⬆️  Publishing to origin\n")
			if err := gitutils.RunGitCommand("push", "-u", "origin", branchName); err != nil {
				return fmt.Errorf("failed to push branch to origin: %w", err)
			}
		}

		fmt.Printf("\n✨ Branch created!\n")
		fmt.Printf("   %s\n", branchName)
		if doPush {
			fmt.Println("   🔗 Published to origin")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().Bool("push", false, "Immediately push the new branch to the remote")
}
