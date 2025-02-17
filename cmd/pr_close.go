package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/crazywolf132/sage/internal/app"
	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var prCloseCmd = &cobra.Command{
	Use:   "close [pr-num]",
	Short: "Close a PR without merging",
	Long: `Close a pull request without merging it.
If no PR number is provided, attempts to close the PR for the current branch.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ghc := gh.NewClient()
		g := git.NewShellGit()

		var num int
		var err error

		if len(args) == 1 {
			// PR number provided
			num, err = strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %v", err)
			}
		} else {
			// Use current branch's PR
			branch, err := g.CurrentBranch()
			if err != nil {
				return err
			}

			// Check if we're on main/master branch
			defaultBranch, _ := g.DefaultBranch()
			if branch == defaultBranch {
				return fmt.Errorf("You're on the %s branch which typically doesn't have a PR.\nTo close a specific PR, provide its number: sage pr close <number>", defaultBranch)
			}

			pr, err := ghc.GetPRForBranch(branch)
			if err != nil {
				if strings.Contains(err.Error(), "Bad credentials") || strings.Contains(err.Error(), "401") {
					return fmt.Errorf("GitHub authentication failed.\n\nTo authenticate, either:\n1. Login with GitHub CLI: gh auth login\n2. Create a token: https://github.com/settings/tokens/new?scopes=repo&description=sage\n3. Set the token:\n   - Run: sage config set github_token YOUR_TOKEN\n   - Or set GITHUB_TOKEN environment variable")
				}
				return err
			}
			if pr == nil {
				return fmt.Errorf("No open PR found for branch '%s'.\n\nTo create a PR: sage pr create\nTo close a specific PR: sage pr close <number>\nTo list open PRs: sage pr list", branch)
			}
			num = pr.Number
		}

		// Get PR details to check status
		pr, err := ghc.GetPRDetails(num)
		if err != nil {
			if strings.Contains(err.Error(), "Bad credentials") || strings.Contains(err.Error(), "401") {
				return fmt.Errorf("GitHub authentication failed.\n\nTo authenticate, either:\n1. Login with GitHub CLI: gh auth login\n2. Create a token: https://github.com/settings/tokens/new?scopes=repo&description=sage\n3. Set the token:\n   - Run: sage config set github_token YOUR_TOKEN\n   - Or set GITHUB_TOKEN environment variable")
			}
			return fmt.Errorf("failed to get PR details: %w", err)
		}

		// Check if PR is already closed or merged
		if pr.State == "closed" {
			return fmt.Errorf("PR #%d is already closed", num)
		}
		if pr.Merged {
			return fmt.Errorf("PR #%d is already merged", num)
		}

		if err := app.ClosePR(ghc, num); err != nil {
			if strings.Contains(err.Error(), "Bad credentials") || strings.Contains(err.Error(), "401") {
				return fmt.Errorf("GitHub authentication failed.\n\nTo authenticate, either:\n1. Login with GitHub CLI: gh auth login\n2. Create a token: https://github.com/settings/tokens/new?scopes=repo&description=sage\n3. Set the token:\n   - Run: sage config set github_token YOUR_TOKEN\n   - Or set GITHUB_TOKEN environment variable")
			}
			return err
		}

		fmt.Printf("%s Closed PR #%d\n", ui.Green("✓"), num)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCloseCmd)
}
