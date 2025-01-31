package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
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
	Short: "Create a new PR on GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		ghc := gh.NewClient() // uses GH_TOKEN or etc

		pr, err := app.CreatePullRequest(g, ghc, app.CreatePROpts{
			Title: prTitle,
			Body:  prBody,
			Base:  prBase,
			Draft: prDraft,
		})
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
}
