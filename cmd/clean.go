package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up merged branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		info, err := app.FindCleanableBranches(g)
		if err != nil {
			return err
		}
		if len(info.Branches) == 0 {
			fmt.Println(ui.Green("No merged branches to clean."))
			return nil
		}

		fmt.Println(ui.Bold("Merged branches that can be deleted:"))
		for _, br := range info.Branches {
			fmt.Printf("  %s\n", br)
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

		results := app.DeleteLocalBranches(g, info.Branches)
		for _, r := range results {
			if r.Err != nil {
				fmt.Printf("%s Could not delete %s: %v\n", ui.Red("✗"), r.Branch, r.Err)
			} else {
				fmt.Printf("%s Deleted %s\n", ui.Green("✓"), r.Branch)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
