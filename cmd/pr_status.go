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

var prStatusCmd = &cobra.Command{
	Use:   "status [pr-num]",
	Short: "Show PR status details (uses current branch's PR if no number specified)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ghc := gh.NewClient()
		g := git.NewShellGit()

		var num int
		var err error

		if len(args) == 1 {
			// PR number provided
			num, err = strconv.Atoi(args[0])
			if err != nil {
				return err
			}
		} else {
			// Use current branch's PR
			branch, err := g.CurrentBranch()
			if err != nil {
				return err
			}

			// List PRs for this branch
			prs, err := ghc.ListPRs("open")
			if err != nil {
				return err
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
				return fmt.Errorf("no PR number provided and no PR found for current branch %q", branch)
			}
		}

		details, err := app.GetPRDetails(ghc, num)
		if err != nil {
			return err
		}

		fmt.Printf("%s PR #%d: %s\n", ui.Sage("ℹ"), details.Number, details.Title)
		fmt.Printf("   URL: %s\n   State: %s\n", details.HTMLURL, details.State)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prStatusCmd)
}
