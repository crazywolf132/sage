package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show repository status",
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()
		st, err := app.GetRepoStatus(g)
		if err != nil {
			return err
		}
		fmt.Printf("%s Current branch: %s\n", ui.Sage("•"), st.Branch)
		if len(st.Changes) == 0 {
			fmt.Println(ui.Green("Working directory is clean."))
			return nil
		}
		fmt.Println("Changes:")
		for _, c := range st.Changes {
			fmt.Printf(" %s %s (%s)\n", c.Symbol, c.File, c.Description)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
