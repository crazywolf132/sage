package cmd

import (
	"errors"
	"fmt"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/spf13/cobra"
)

var listState string

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests in the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := githubutils.GetGitHubToken()
		if err != nil {
			return err
		}
		if token == "" {
			return errors.New("no GitHub token found; install GH CLI or set SAGE_GITHUB_TOKEN / GITHUB_TOKEN")
		}

		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			return err
		}

		if listState == "" {
			listState = "open" // default to open
		}

		prs, err := githubutils.ListPullRequests(token, owner, repo, listState)
		if err != nil {
			return err
		}

		for _, pr := range prs {
			fmt.Printf("#%d [%s] %s\n", pr.Number, pr.State, pr.Title)
		}
		return nil
	},
}

func init() {
	prCmd.AddCommand(prListCmd)
	prListCmd.Flags().StringVar(&listState, "state", "open", "Filter by state: open, closed, or all")
}
