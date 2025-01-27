package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/spf13/cobra"
)

var mergeMethod string

var prMergeCmd = &cobra.Command{
	Use:   "merge <pr-number>",
	Short: "Merge a pull request (default merge method: 'merge')",
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

		if mergeMethod == "" {
			mergeMethod = "merge" // "merge", "squash", or "rebase"
		}

		if err := githubutils.MergePullRequest(token, owner, repo, prNumber, mergeMethod); err != nil {
			return err
		}

		fmt.Printf("PR #%d merged using '%s' method.\n", prNumber, mergeMethod)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prMergeCmd)
	prMergeCmd.Flags().StringVar(&mergeMethod, "method", "merge", "Merge method: merge, squash, rebase")
}
