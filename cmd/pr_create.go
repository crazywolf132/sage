package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

var (
	prTitle     string
	prBody      string
	prBase      string
	prDraft     bool
	prReviewers []string
	prLabels    []string
	useTUI      bool
	prUseAI     bool
)

// prCreateCmd is "sage pr create"
var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new PR on GitHub (interactive if flags not provided)",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		ghc := gh.NewClient()

		// If AI flag is set, generate PR content first
		if prUseAI {
			aiForm, err := ui.GenerateAIPRContent(g, ghc)
			if err != nil {
				return fmt.Errorf("failed to generate AI content: %w", err)
			}
			prTitle = aiForm.Title
			prBody = aiForm.Body
			// Don't override other flags if they were explicitly set
			if len(prLabels) == 0 {
				prLabels = aiForm.Labels
			}
			if len(prReviewers) == 0 {
				prReviewers = aiForm.Reviewers
			}
		}

		// Show interactive form if required fields are missing
		if prTitle == "" || prBody == "" {
			form, err := ui.AskPRForm(ui.PRForm{
				Title:     prTitle,
				Body:      prBody,
				Base:      prBase,
				Draft:     prDraft,
				Labels:    prLabels,
				Reviewers: prReviewers,
			}, ghc)
			if err != nil {
				return err
			}
			// copy back
			prTitle = form.Title
			prBody = form.Body
			prBase = form.Base
			prDraft = form.Draft
			prLabels = form.Labels
			prReviewers = form.Reviewers
		}

		opts := app.CreatePROpts{
			Title:     prTitle,
			Body:      prBody,
			Base:      prBase,
			Draft:     prDraft,
			Labels:    prLabels,
			Reviewers: prReviewers,
		}
		pr, err := app.CreatePullRequest(g, ghc, opts)
		if err != nil {
			return err
		}

		fmt.Printf("%s Created PR #%d: %s\n", ui.Green("✓"), pr.Number, pr.HTMLURL)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCreateCmd)

	prCreateCmd.Flags().StringVarP(&prTitle, "title", "t", "", "PR Title")
	prCreateCmd.Flags().StringVarP(&prBody, "body", "b", "", "PR Body")
	prCreateCmd.Flags().StringVar(&prBase, "base", "", "Base branch (default=main)")
	prCreateCmd.Flags().BoolVar(&prDraft, "draft", false, "Create as draft PR")
	prCreateCmd.Flags().StringSliceVar(&prReviewers, "reviewer", nil, "Add one or more reviewers")
	prCreateCmd.Flags().StringSliceVar(&prLabels, "label", nil, "Add one or more labels")
	prCreateCmd.Flags().BoolVarP(&prUseAI, "ai", "a", false, "Use AI to generate PR content")
}
