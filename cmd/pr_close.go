package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/spf13/cobra"
)

var prCloseCmd = &cobra.Command{
	Use:   "close <pr-number>",
	Short: "Close a pull request without merging",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := githubutils.GetGitHubToken()
		if err != nil {
			return err
		}
		if token == "" {
			return errors.New("no GitHub token found")
		}

		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			return err
		}

		prNumber, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}

		if err := githubutils.ClosePullRequest(token, owner, repo, prNumber); err != nil {
			return err
		}

		fmt.Printf("PR #%d closed.\n", prNumber)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCloseCmd)
}
