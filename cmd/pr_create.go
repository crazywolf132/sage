package cmd

import (
	"errors"
	"fmt"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	prTitle string
	prBody  string
	prBase  string
	prDraft bool
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pull request on GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. get token
		token, err := githubutils.GetGitHubToken()
		if err != nil {
			return err
		}
		if token == "" {
			return errors.New("no GitHub token found; install GH CLI or set SAGE_GITHUB_TOKEN / GITHUB_TOKEN")
		}

		// 2. owner, repo
		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			return err
		}

		// 3. current branch
		currentBranch, err := gitutils.GetCurrentBranch()
		if err != nil {
			return err
		}

		// If title is not provided via flag, use the interactive form
		if prTitle == "" {
			form, err := ui.GetPRDetails()
			if err != nil {
				return err
			}
			prTitle = form.Title
			if prBody == "" {
				prBody = form.Body
			}
		}

		// default base if none provided
		if prBase == "" {
			prBase = "main"
		}

		// 4. build create params
		prParams := githubutils.CreatePRParams{
			Title: prTitle,
			Head:  currentBranch,
			Base:  prBase,
			Body:  prBody,
			Draft: prDraft,
		}

		// 5. make the API call
		pr, err := githubutils.CreatePullRequest(token, owner, repo, prParams)
		if err != nil {
			return err
		}

		fmt.Printf("Pull Request created! #%d\nURL: %s\n", pr.Number, pr.HTMLURL)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCreateCmd)

	prCreateCmd.Flags().StringVarP(&prTitle, "title", "t", "", "Title of the pull request")
	prCreateCmd.Flags().StringVarP(&prBody, "body", "b", "", "Body/description of the pull request")
	prCreateCmd.Flags().StringVar(&prBase, "base", "", "Base branch (default: main)")
	prCreateCmd.Flags().BoolVar(&prDraft, "draft", false, "Create a draft pull request")
}
