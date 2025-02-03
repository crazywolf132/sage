package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
)

func SquashCommits(g git.Service, startCommit string, all bool) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repo")
	}

	// Get current branch
	currentBranch, err := g.CurrentBranch()
	if err != nil {
		return fmt.Errorf("failed to get current branch: %w", err)
	}

	// Check if we're on the head branch
	isHead, err := g.IsHeadBranch(currentBranch)
	if err != nil {
		return fmt.Errorf("failed to check if current branch is head: %w", err)
	}

	// Don't allow squashing on head branch unless explicitly allowed
	if isHead && all {
		return fmt.Errorf("cannot squash all commits on the head branch")
	}

	// If all flag is set, get the first commit
	if all {
		firstCommit, err := g.GetFirstCommit()
		if err != nil {
			return fmt.Errorf("failed to get first commit: %w", err)
		}
		startCommit = firstCommit
	}

	// Ensure we have a start commit
	if startCommit == "" {
		return fmt.Errorf("no start commit specified")
	}

	// Start interactive rebase
	if err := g.SquashCommits(startCommit); err != nil {
		return fmt.Errorf("failed to start interactive rebase: %w", err)
	}

	return nil
}
