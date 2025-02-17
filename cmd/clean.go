package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cleanNoRemote bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up merged branches",
	Long: `Clean up branches that have been merged or whose PRs have been closed.
This includes:
- Branches that are merged into the default branch
- Branches whose PRs were closed or merged through GitHub
- Remote branches that were deleted`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		ghc := gh.NewClient()

		info, err := app.FindCleanableBranches(g, ghc)
		if err != nil {
			return err
		}

		if len(info.LocalBranches) == 0 && len(info.RemoteBranches) == 0 {
			fmt.Println(ui.Green("No branches to clean."))
			return nil
		}

		// Show branches that will be deleted
		if len(info.LocalBranches) > 0 {
			fmt.Println(ui.Bold("\nLocal branches to delete:"))
			for _, br := range info.LocalBranches {
				fmt.Printf("  %s\n", br)
			}
		}

		if !cleanNoRemote && len(info.RemoteBranches) > 0 {
			fmt.Println(ui.Bold("\nRemote branches to delete:"))
			for _, br := range info.RemoteBranches {
				fmt.Printf("  origin/%s\n", br)
			}
		}

		var confirm bool
		prompt := &survey.Confirm{
			Message: "Delete these branches?",
		}
		if err := survey.AskOne(prompt, &confirm); err != nil {
			return err
		}
		if !confirm {
			fmt.Println(ui.Gray("Aborted."))
			return nil
		}

		// Delete local branches
		if len(info.LocalBranches) > 0 {
			results := app.DeleteLocalBranches(g, info.LocalBranches)
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%s Could not delete %s: %v\n", ui.Red("✗"), r.Branch, r.Err)
				} else {
					fmt.Printf("%s Deleted local branch %s\n", ui.Green("✓"), r.Branch)
				}
			}
		}

		// Delete remote branches
		if !cleanNoRemote && len(info.RemoteBranches) > 0 {
			results := app.DeleteRemoteBranches(g, info.RemoteBranches)
			for _, r := range results {
				if r.Err != nil {
					fmt.Printf("%s Could not delete %s: %v\n", ui.Red("✗"), r.Branch, r.Err)
				} else {
					fmt.Printf("%s Deleted remote branch %s\n", ui.Green("✓"), r.Branch)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVar(&cleanNoRemote, "no-remote", false, "Skip deleting remote branches")
}
