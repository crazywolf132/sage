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

		if len(args) == 0 {
			// No PR number provided, try to find PR for current branch
			branch, err := g.CurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}

			// Get open PRs
			prs, err := ghc.ListPRs("open")
			if err != nil {
				return fmt.Errorf("failed to list PRs: %w", err)
			}

			// Find PR for current branch
			var found bool
			for _, pr := range prs {
				if pr.Head.Ref == branch {
					num = pr.Number
					found = true
					break
				}
			}

			if !found {
				return fmt.Errorf("no open PR found for current branch %q", branch)
			}

			fmt.Printf("%s Found PR #%d for current branch\n", ui.Sage("ℹ"), num)
		} else {
			// PR number provided
			num, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number %q: %w", args[0], err)
			}
		}

		if method == "" {
			method = "merge"
		}

		if err := app.MergePR(ghc, num, method); err != nil {
			return err
		}

		fmt.Printf("%s Merged PR #%d with method=%s\n", ui.Green("✓"), num, method)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prMergeCmd)
	prMergeCmd.Flags().StringVar(&method, "method", "merge", "merge|squash|rebase")
}
