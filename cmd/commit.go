package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var (
	commitMessage      string
	commitEmpty        bool
	commitPush         bool
	commitConventional bool
	commitAI           bool
	commitAutoAccept   bool
	commitAmend        bool
	commitOnlyStaged   bool
	commitInteractive  bool
)

var commitCmd = &cobra.Command{
	Use:   "commit [message]",
	Short: "Stage and commit changes",
	Long: `Stage and commit changes in one step.

Examples:
  # Interactive commit message prompt with AI support
  sage commit --ai

  # Direct commit with message
  sage commit "feat: add user authentication"

  # Stage everything (including .sage/) and commit with push
  sage commit -p "fix: resolve null pointer error"

  # Commit only staged changes (useful after using 'sage stage')
  sage commit -s "chore: update dependencies"

  # Interactive selection of files to stage during commit
  sage commit -i "docs: update readme"
  
  # Amend the last commit with updated files or commit message
  sage commit --amend "refactor: update commit message"
  
When files have been manually staged with 'git add' or 'sage stage', 
Sage will detect this and give you smart options to either:
  - Commit only the staged changes
  - Stage everything and commit
  - View what's staged vs unstaged before deciding`,
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"c"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			commitMessage = args[0]
		}

		g := git.NewShellGit()

		res, err := app.Commit(g, app.CommitOptions{
			Message:         commitMessage,
			AllowEmpty:      commitEmpty,
			PushAfterCommit: commitPush,
			UseConventional: commitConventional,
			UseAI:           commitAI,
			AutoAccept:      commitAutoAccept,
			Amend:           commitAmend,
			OnlyStaged:      commitOnlyStaged,
			Interactive:     commitInteractive,
		})
		if err != nil {
			return err
		}

		fmt.Printf("%s Created commit", ui.Green("✓"))
		if res.Pushed {
			fmt.Printf(" and pushed")
		}
		fmt.Printf(": %s\n", res.ActualMessage)

		// Display file statistics
		stats := res.Stats
		if stats.TotalStaged > 0 {
			fmt.Println()
			fmt.Println(ui.Bold("Files committed:"))
			if stats.StagedAdded > 0 {
				fmt.Printf("  %s added\n", ui.Green(fmt.Sprintf("%d", stats.StagedAdded)))
			}
			if stats.StagedModified > 0 {
				fmt.Printf("  %s modified\n", ui.Yellow(fmt.Sprintf("%d", stats.StagedModified)))
			}
			if stats.StagedDeleted > 0 {
				fmt.Printf("  %s deleted\n", ui.Red(fmt.Sprintf("%d", stats.StagedDeleted)))
			}
			fmt.Printf("%s %d files changed\n", ui.Bold("Total:"), stats.TotalStaged)
		}

		// Remind user about unstaged files if any
		if stats.TotalUnstaged > 0 {
			fmt.Printf("\n%s You still have %d unstaged files. Use %s to commit them.\n",
				ui.Yellow("!"),
				stats.TotalUnstaged,
				ui.White("sage commit"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().BoolVar(&commitEmpty, "empty", false, "Allow empty commits")
	commitCmd.Flags().BoolVarP(&commitPush, "push", "p", false, "Push after commit")
	commitCmd.Flags().BoolVarP(&commitConventional, "conventional", "c", false, "Use conventional commit format")
	commitCmd.Flags().BoolVarP(&commitAI, "ai", "a", false, "Use AI to generate commit message")
	commitCmd.Flags().BoolVarP(&commitAutoAccept, "yes", "y", false, "Automatically accept AI-generated commit message")
	commitCmd.Flags().BoolVar(&commitAmend, "amend", false, "Amend the last commit")
	commitCmd.Flags().BoolVarP(&commitOnlyStaged, "only-staged", "s", false, "Commit only staged changes (don't automatically stage all files)")
	commitCmd.Flags().BoolVarP(&commitInteractive, "interactive", "i", false, "Interactively select files to commit")
	commitCmd.Flags().StringVarP(&commitMessage, "message", "m", "", "Commit message")
}
