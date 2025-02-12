package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

var (
	prMergeMethod string
)

// parsePRNumber converts a string PR number to int
func parsePRNumber(s string) (int, error) {
	num, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid PR number %q: %w", s, err)
	}
	if num <= 0 {
		return 0, fmt.Errorf("PR number must be positive, got %d", num)
	}
	return num, nil
}

// prMergeCmd represents the "sage pr merge" command
var prMergeCmd = &cobra.Command{
	Use:   "merge [pr-number]",
	Short: "Merge a pull request",
	Long: `Merge a pull request. If no PR number is provided, attempts to merge the PR for the current branch.
Supports different merge methods: merge (default), squash, or rebase.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		ghc := gh.NewClient()

		var prNum int
		var err error

		if len(args) > 0 {
			// Parse PR number from args
			if prNum, err = parsePRNumber(args[0]); err != nil {
				return err
			}
		} else {
			// Try to find PR for current branch
			branch, err := g.CurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}

			pr, err := ghc.GetPRForBranch(branch)
			if err != nil {
				return fmt.Errorf("failed to get PR for branch: %w", err)
			}
			if pr == nil {
				return fmt.Errorf("no pull request found for branch '%s'", branch)
			}
			prNum = pr.Number
			fmt.Printf("ℹ Found PR #%d for current branch\n", prNum)
		}

		// Get PR details to check status
		pr, err := ghc.GetPRDetails(prNum)
		if err != nil {
			return fmt.Errorf("failed to get PR details: %w", err)
		}

		// Check if PR is already merged
		if pr.Merged {
			return fmt.Errorf("PR #%d is already merged", prNum)
		}

		// Check if PR is closed
		if pr.State == "closed" {
			return fmt.Errorf("PR #%d is closed", prNum)
		}

		// Try to merge with the specified method
		if err := app.MergePR(ghc, prNum, prMergeMethod); err != nil {
			// Check for specific error cases and provide helpful messages
			if err.Error() == "merge commits are not allowed on this repository" {
				fmt.Printf("ℹ Merge commits are not allowed. Trying squash merge instead...\n")
				if err := app.MergePR(ghc, prNum, "squash"); err != nil {
					return fmt.Errorf("failed to squash merge: %w", err)
				}
			} else {
				return fmt.Errorf("failed to merge PR: %w", err)
			}
		}

		fmt.Printf("✓ Successfully merged PR #%d\n", prNum)

		// Cleanup: delete the local branch if we're on it
		currentBranch, _ := g.CurrentBranch()
		if currentBranch == pr.Head.Ref {
			defaultBranch, err := g.DefaultBranch()
			if err != nil {
				defaultBranch = "main" // fallback
			}

			// Switch to default branch
			if err := g.RunInteractive("switch", defaultBranch); err != nil {
				fmt.Printf("⚠ Failed to switch to %s branch: %v\n", defaultBranch, err)
				return nil
			}

			// Delete the merged branch
			if err := g.RunInteractive("branch", "-D", currentBranch); err != nil {
				fmt.Printf("⚠ Failed to delete local branch: %v\n", err)
				return nil
			}

			fmt.Printf("✓ Cleaned up local branch '%s'\n", currentBranch)
		}

		return nil
	},
}

func init() {
	prCmd.AddCommand(prMergeCmd)
	prMergeCmd.Flags().StringVarP(&prMergeMethod, "method", "m", "merge", "Merge method: merge, squash, or rebase")
}
