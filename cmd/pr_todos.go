package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/crazywolf132/sage/internal/githubutils"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prTodosCmd = &cobra.Command{
	Use:   "todos [pr-number]",
	Short: "Show unresolved comment threads in a pull request",
	Long: `Display all unresolved comment threads in a pull request.
This helps track what still needs to be addressed before the PR can be merged.
If no PR number is provided, it will look for a PR associated with the current branch.

You can configure bots to ignore using:
  git config sage.ignore-bots "github-actions[bot],dependabot[bot]"`,
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

		// Get list of bots to ignore
		botsToIgnore := make(map[string]bool)
		if botsConfig, err := gitutils.DefaultRunner.RunGitCommandWithOutput("config", "--get", "sage.ignore-bots"); err == nil {
			for _, bot := range strings.Split(strings.TrimSpace(botsConfig), ",") {
				botsToIgnore[strings.TrimSpace(bot)] = true
			}
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
			// Skip comments from ignored bots
			if botsToIgnore[comment.User.Login] {
				continue
			}
			threadKey := comment.Path + ":" + strconv.Itoa(comment.Line)
			if comment.ThreadID != "" {
				threadKey = comment.ThreadID
			}
			threads[threadKey] = append(threads[threadKey], comment)
		}

		// Print unresolved threads
		hasUnresolved := false
		for _, thread := range threads {
			// Skip empty threads (could happen if all comments were from bots)
			if len(thread) == 0 {
				continue
			}

			// Check if thread is resolved
			lastComment := thread[len(thread)-1]
			// Skip if last comment is from an ignored bot
			if botsToIgnore[lastComment.User.Login] {
				continue
			}
			if !lastComment.Resolved && !isResolutionComment(lastComment.Body) {
				if !hasUnresolved {
					fmt.Printf("\n%s\n", ui.ColoredText("Unresolved Threads:", ui.Sage))
					hasUnresolved = true
				}

				// Print thread location
				firstComment := thread[0]
				if firstComment.Path != "" {
					fmt.Printf("\n%s %s:%d\n",
						ui.ColoredText("→", ui.Yellow),
						ui.ColoredText(firstComment.Path, ui.Blue),
						firstComment.Line)
				} else {
					fmt.Printf("\n%s %s\n",
						ui.ColoredText("→", ui.Yellow),
						ui.ColoredText("General Comment", ui.Blue))
				}

				// Print thread comments
				for _, comment := range thread {
					// Skip comments from ignored bots
					if botsToIgnore[comment.User.Login] {
						continue
					}
					timestamp := comment.CreatedAt.Format("Jan 02")
					fmt.Printf("  %s %s: %s\n",
						ui.ColoredText(timestamp, ui.White),
						ui.ColoredText("@"+comment.User.Login, ui.Yellow),
						strings.Split(comment.Body, "\n")[0]) // Show first line only
				}
			}
		}

		if !hasUnresolved {
			fmt.Println("\n✨ No unresolved comment threads!")
		}

		return nil
	},
}

// isResolutionComment checks if a comment appears to be resolving the thread
func isResolutionComment(body string) bool {
	body = strings.ToLower(body)
	resolutionPhrases := []string{
		"fixed",
		"done",
		"resolved",
		"addressed",
		"thanks",
		"thank you",
		"👍",
		"lgtm",
	}
	for _, phrase := range resolutionPhrases {
		if strings.Contains(body, phrase) {
			return true
		}
	}
	return false
}

func init() {
	prCmd.AddCommand(prTodosCmd)
}
