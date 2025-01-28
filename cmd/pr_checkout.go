package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/spf13/cobra"
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <pr-number>",
	Short: "Check out a PR branch locally",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return errors.New("invalid PR number")
		}

		token, err := githubutils.GetGitHubToken()
		if err != nil {
			return fmt.Errorf("failed to get GitHub token: %w", err)
		}

		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			return fmt.Errorf("failed to get repo info: %w", err)
		}

		err = githubutils.CheckoutPullRequest(token, owner, repo, prNumber)
		if err != nil {
			return fmt.Errorf("failed to checkout PR %d: %w", prNumber, err)
		}

		fmt.Printf("Checked out PR #%d as 'pr-%d'\n", prNumber, prNumber)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCheckoutCmd)
}
