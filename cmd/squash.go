package cmd

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	squashAll bool
)

var squashCmd = &cobra.Command{
	Use:   "squash [commit]",
	Short: "Squash commits interactively",
	Long: `Squash commits using interactive rebase.
If --all flag is used, squashes all commits in the branch (except on head branch).
Otherwise, squashes from the specified commit or prompts for selection.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		var startCommit string
		if len(args) == 1 {
			startCommit = args[0]
		} else if !squashAll {
			// Get commit history for selection
			history, err := g.Log("HEAD", 10, false, false)
			if err != nil {
				return fmt.Errorf("failed to get commit history: %w", err)
			}

			// Parse commit history
			commits := make([]string, 0)
			for _, line := range strings.Split(strings.TrimSpace(history), "\n") {
				parts := strings.Split(line, "\x00")
				if len(parts) >= 4 {
					commits = append(commits, fmt.Sprintf("%s %s", parts[0][:8], parts[3]))
				}
			}

			if len(commits) == 0 {
				return fmt.Errorf("no commits found")
			}

			// Prompt for commit selection
			var selected string
			prompt := &survey.Select{
				Message: "Select commit to squash from:",
				Options: commits,
			}
			if err := survey.AskOne(prompt, &selected); err != nil {
				return err
			}

			// Extract commit hash
			startCommit = strings.Split(selected, " ")[0]
		}

		if err := app.SquashCommits(g, startCommit, squashAll); err != nil {
			return err
		}

		fmt.Printf("%s Started interactive rebase for squashing commits\n", ui.Green("✓"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(squashCmd)
	squashCmd.Flags().BoolVarP(&squashAll, "all", "a", false, "Squash all commits in the branch")
}
