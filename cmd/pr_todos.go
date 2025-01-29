package cmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prTodosCmd = &cobra.Command{
	Use:   "todos [pr-number]",
	Short: "Show unresolved comment threads in a pull request",
	Long: `Display all unresolved comment threads in a pull request.
This helps track what still needs to be addressed before the PR can be merged.
If no PR number is provided, it will look for a PR associated with the current branch.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. Get token
		token, err := githubutils.GetGitHubToken()
		if err != nil {
			return err
		}
		if token == "" {
			return errors.New("no GitHub token found; install GH CLI or set SAGE_GITHUB_TOKEN / GITHUB_TOKEN")
		}

		// 2. Get owner/repo
		owner, repo, err := githubutils.FindRepoOwnerAndName()
		if err != nil {
			return err
		}

		var prNumber int
		if len(args) == 1 {
			// If PR number provided, use it
			prNumber, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %s", args[0])
			}
		} else {
			// If no PR number, try to get current branch's PR
			currentPR, err := githubutils.GetCurrentBranchPR(token, owner, repo)
			if err != nil {
				return fmt.Errorf("failed to get PR for current branch: %w", err)
			}
			if currentPR == nil {
				return errors.New("no pull request found for current branch")
			}
			prNumber = currentPR.Number
		}

		// Get PR review comments
		comments, err := githubutils.GetPRReviewComments(token, owner, repo, prNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR comments: %w", err)
		}

		// Group comments by thread
		threads := make(map[string][]githubutils.PRReviewComment)
		for _, comment := range comments {
			if comment.ThreadID != "" {
				threads[comment.ThreadID] = append(threads[comment.ThreadID], comment)
			}
		}

		// Print unresolved threads
		hasUnresolved := false
		for _, thread := range threads {
			// Check if thread is resolved
			lastComment := thread[len(thread)-1]
			if !lastComment.Resolved {
				if !hasUnresolved {
					fmt.Printf("\n%s\n", ui.ColoredText("Unresolved Threads:", ui.Sage))
					hasUnresolved = true
				}

				// Print thread location
				firstComment := thread[0]
				fmt.Printf("\n%s %s:%d\n",
					ui.ColoredText("→", ui.Yellow),
					ui.ColoredText(firstComment.Path, ui.Blue),
					firstComment.Line)

				// Print thread comments
				for _, comment := range thread {
					timestamp := comment.CreatedAt.Format("Jan 02")
					fmt.Printf("  %s %s: %s\n",
						ui.ColoredText(timestamp, ui.White),
						ui.ColoredText("@"+comment.User.Login, ui.Yellow),
						comment.Body)
				}
			}
		}

		if !hasUnresolved {
			fmt.Println("\n✨ No unresolved comment threads!")
		}

		return nil
	},
}

func init() {
	prCmd.AddCommand(prTodosCmd)
}
