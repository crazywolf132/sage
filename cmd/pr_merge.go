package cmd

import (
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var method string

var prMergeCmd = &cobra.Command{
	Use:   "merge [pr-num]",
	Short: "Merge a pull request (uses PR from current branch if no number provided)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ghc := gh.NewClient()
		g := git.NewShellGit()

		var num int
		var err error
		var currentBranch string

		// Get current branch first as we'll need it later
		currentBranch, err = g.CurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		if len(args) == 0 {
			// No PR number provided, try to find PR for current branch
			// Get open PRs
			prs, err := ghc.ListPRs("open")
			if err != nil {
				return fmt.Errorf("failed to list PRs: %w", err)
			}

			// Find PR for current branch
			var found bool
			for _, pr := range prs {
				if pr.Head.Ref == currentBranch {
					num = pr.Number
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("no open PR found for current branch %q", currentBranch)
			}

			fmt.Printf("%s Found PR #%d for current branch\n", ui.Sage("ℹ"), num)
		} else {
			// PR number provided
			num, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number %q: %w", args[0], err)
			}
		}

		// Get PR details to know the base branch
		pr, err := ghc.GetPRDetails(num)
		if err != nil {
			return fmt.Errorf("failed to get PR details: %w", err)
		}

		if method == "" {
			method = "merge"
		}

		if err := app.MergePR(ghc, num, method); err != nil {
			return err
		}

		fmt.Printf("%s Merged PR #%d with method=%s\n", ui.Green("✓"), num, method)

		// If we're on the PR branch, switch back to base branch
		if currentBranch == pr.Head.Ref {
			fmt.Printf("%s Switching back to base branch %q\n", ui.Sage("ℹ"), pr.Base.Ref)
			if err := g.Checkout(pr.Base.Ref); err != nil {
				return fmt.Errorf("failed to switch to base branch: %w", err)
			}

			// Pull latest changes
			if err := g.Pull(); err != nil {
				return fmt.Errorf("failed to pull latest changes: %w", err)
			}
			fmt.Printf("%s Switched to %s and pulled latest changes\n", ui.Green("✓"), pr.Base.Ref)
		}

		return nil
	},
}

func init() {
	prCmd.AddCommand(prMergeCmd)
	prMergeCmd.Flags().StringVar(&method, "method", "merge", "merge|squash|rebase")
}
