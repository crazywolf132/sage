package cmd

import (
	"errors"
	"fmt"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	prTitle     string
	prBody      string
	prBase      string
	prDraft     bool
	useTemplate bool
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
			// Try to get the first commit message as a title placeholder
			firstCommit, err := gitutils.GetFirstCommitOnBranch()
			if err != nil {
				fmt.Println("Warning: Failed to get first commit message:", err)
			}

			// Get PR template if requested and available
			var templateContent string
			if useTemplate {
				templateContent, err = githubutils.GetPullRequestTemplate(token, owner, repo)
				if err != nil {
					fmt.Println("Warning: Failed to fetch PR template:", err)
				}
			}

			// Try to load backup if it exists
			backup, err := ui.LoadPRFormBackup()
			if err != nil {
				fmt.Println("Warning: Failed to load PR form backup:", err)
			}

			// Pre-populate the form with backup, template, or first commit
			form := ui.PRForm{
				Title: firstCommit, // Use first commit as default title
				Body:  templateContent,
			}
			if backup != nil {
				form = *backup
			}

			// Get PR details through the form
			form, err = ui.GetPRDetails(form.Body)
			if err != nil {
				return err
			}
			prTitle = form.Title
			if prBody == "" {
				prBody = form.Body
			}

			// Backup the form data
			if err := ui.BackupPRForm(form); err != nil {
				fmt.Println("Warning: Failed to backup PR form:", err)
			}
		}

		// default base if none provided
		if prBase == "" {
			prBase = "main"
		}

		// Handle draft PR settings
		if !prDraft { // If not explicitly set to draft via flag
			if viper.GetBool("pr.forceDraft") {
				prDraft = true
			} else if viper.GetBool("pr.defaultDraft") {
				prDraft = true
			}
		}

		// Ensure we have required fields to avoid 422 errors
		if prTitle == "" {
			return fmt.Errorf("PR title cannot be empty")
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
			return fmt.Errorf("failed to create PR: %w", err)
		}

		// Clean up the backup file since PR was created successfully
		if err := ui.DeletePRFormBackup(); err != nil {
			fmt.Println("Warning: Failed to delete PR form backup:", err)
		}

		fmt.Printf("Pull Request created! #%d\nURL: %s\n", pr.Number, pr.HTMLURL)
		if pr.Draft {
			fmt.Println("Note: This PR was created as a draft.")
		}
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCreateCmd)

	prCreateCmd.Flags().StringVarP(&prTitle, "title", "t", "", "Title of the pull request")
	prCreateCmd.Flags().StringVarP(&prBody, "body", "b", "", "Body/description of the pull request")
	prCreateCmd.Flags().StringVar(&prBase, "base", "", "Base branch (default: main)")
	prCreateCmd.Flags().BoolVar(&prDraft, "draft", false, "Create a draft pull request")
	prCreateCmd.Flags().BoolVar(&useTemplate, "template", true, "Use repository's PR template if available")
}
