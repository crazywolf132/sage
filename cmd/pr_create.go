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
			// Try to load backup first
			backup, err := ui.LoadPRFormBackup()
			if err != nil {
				fmt.Println("Warning: Failed to load PR form backup:", err)
			}

			// Get PR template if requested and available
			var templateContent string
			if useTemplate {
				templateContent, err = githubutils.GetPullRequestTemplate(token, owner, repo)
				if err != nil {
					fmt.Println("Warning: Failed to fetch PR template:", err)
				}
			}

			// Initialize form with backup if it exists
			var form ui.PRForm
			if backup != nil {
				form = *backup
			} else {
				// Try to get the first commit message as a title placeholder
				firstCommit, err := gitutils.GetFirstCommitOnBranch()
				if err != nil {
					fmt.Println("Warning: Failed to get first commit message:", err)
				} else {
					form.Title = firstCommit
				}
				form.Body = templateContent
			}

			// Get PR details through the form
			form, err = ui.GetPRDetails(form)
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
			defaultBranch, err := gitutils.GetDefaultBranch()
			if err != nil {
				fmt.Println("Warning: Failed to determine default branch:", err)
				// Fallback to main if we can't determine default branch
				defaultBranch = "main"
			}
			prBase = defaultBranch
		}

		// Handle draft PR settings
		if !prDraft { // If not explicitly set to draft via flag
			if viper.GetBool("pr.forceDraft") {
				prDraft = true
			} else if viper.GetBool("pr.defaultDraft") {
				prDraft = true
			}
		}

		// Validate required fields
		if prTitle == "" {
			return fmt.Errorf("PR title cannot be empty")
		}
		if currentBranch == "" {
			return fmt.Errorf("current branch cannot be empty")
		}
		if prBase == "" {
			return fmt.Errorf("base branch cannot be empty")
		}

		// Ensure all changes are pushed before creating PR
		fmt.Printf("Pushing latest changes to %s...\n", currentBranch)
		if err := gitutils.RunGitCommand("push", "-u", "origin", currentBranch); err != nil {
			return fmt.Errorf("failed to push changes: %w", err)
		}

		// Build create params
		prParams := githubutils.CreatePRParams{
			Title: prTitle,
			Head:  currentBranch,
			Base:  prBase,
			Body:  prBody,
			Draft: prDraft,
		}

		// Create the pull request
		fmt.Println("Creating pull request...")
		pr, err := githubutils.CreatePullRequest(token, owner, repo, prParams)
		if err != nil {
			return fmt.Errorf("failed to create PR: %w", err)
		}

		// Clean up the backup file since PR was created successfully
		if err := ui.DeletePRFormBackup(); err != nil {
			fmt.Println("Warning: Failed to delete PR form backup:", err)
		}

		fmt.Printf("\n✨ Pull Request #%d created successfully!\n", pr.Number)
		fmt.Printf("🔗 %s\n", pr.HTMLURL)
		if pr.Draft {
			fmt.Println("📝 Created as draft")
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
