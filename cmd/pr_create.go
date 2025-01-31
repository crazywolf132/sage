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
	useTUI      bool
	useTemplate bool
)

var (
	prTitle     string
	prBody      string
	prBase      string
	prDraft     bool
	prReviewers []string
	prLabels    []string
)

// prCreateCmd is "sage pr create"
var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new PR on GitHub (with optional TUI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		ghc := gh.NewClient()

		if useTUI {
			form, err := ui.AskPRForm(ui.PRForm{
				Title:       prTitle,
				Body:        prBody,
				Base:        prBase,
				Draft:       prDraft,
				Labels:      prLabels,
				Reviewers:   prReviewers,
				UseTemplate: useTemplate,
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
			Title:       prTitle,
			Body:        prBody,
			Base:        prBase,
			Draft:       prDraft,
			Labels:      prLabels,
			Reviewers:   prReviewers,
			UseTemplate: useTemplate,
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
	prCreateCmd.Flags().BoolVar(&useTemplate, "template", true, "Use GitHub PR template if available")
	prCreateCmd.Flags().BoolVar(&useTUI, "interactive", false, "Use TUI to gather PR details")
}
