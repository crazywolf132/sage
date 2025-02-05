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

var (
	updateTitle     string
	updateBody      string
	updateDraft     bool
	updateAI        bool
	updateLabels    []string
	updateReviewers []string
)

var prUpdateCmd = &cobra.Command{
	Use:   "update [pr-num]",
	Short: "Update a pull request's fields",
	Long: `Update various fields of a pull request. If no PR number is provided, uses the current branch's PR.
	
You can update the title, body, draft status, labels, and reviewers. With the --ai flag, 
it will automatically update the PR body and labels based on the latest commits.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ghc := gh.NewClient()
		g := git.NewShellGit()

		var num int
		var err error

		if len(args) == 1 {
			num, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %v", err)
			}
		} else {
			branch, err := g.CurrentBranch()
			if err != nil {
				return err
			}

			pr, err := ghc.GetPRForBranch(branch)
			if err != nil {
				return err
			}
			if pr == nil {
				return fmt.Errorf("no open PR found for branch %s", branch)
			}
			num = pr.Number
		}

		err = app.UpdatePR(ghc, g, num, app.PRUpdateOptions{
			Title:     updateTitle,
			Body:      updateBody,
			Draft:     updateDraft,
			UseAI:     updateAI,
			Labels:    updateLabels,
			Reviewers: updateReviewers,
		})
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", ui.Green(fmt.Sprintf("Successfully updated PR #%d", num)))
		return nil
	},
}

func init() {
	prCmd.AddCommand(prUpdateCmd)

	prUpdateCmd.Flags().StringVarP(&updateTitle, "title", "t", "", "Update the PR title")
	prUpdateCmd.Flags().StringVarP(&updateBody, "body", "b", "", "Update the PR body")
	prUpdateCmd.Flags().BoolVar(&updateDraft, "draft", false, "Set the PR's draft status")
	prUpdateCmd.Flags().BoolVar(&updateAI, "ai", false, "Use AI to update the PR body and labels based on latest commits")
	prUpdateCmd.Flags().StringSliceVar(&updateLabels, "labels", nil, "Update PR labels")
	prUpdateCmd.Flags().StringSliceVar(&updateReviewers, "reviewers", nil, "Update PR reviewers")
}
