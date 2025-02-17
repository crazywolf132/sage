package app

import (
	"fmt"
	"strings"
	"sync"

	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/git"
)

type CleanableBranches struct {
	LocalBranches  []string
	RemoteBranches []string
}

type DeletionResult struct {
	Branch string
	Err    error
}

func FindCleanableBranches(g git.Service, ghc gh.Client) (*CleanableBranches, error) {
	isRepo, err := g.IsRepo()
	if err != nil || !isRepo {
		return nil, err
	}

	// Fetch latest remote info
	if err := g.FetchAll(); err != nil {
		return nil, fmt.Errorf("failed to fetch remote updates: %w", err)
	}

	// Get current and default branches
	db, err := g.DefaultBranch()
	if err != nil {
		db = "main"
	}
	cur, err := g.CurrentBranch()
	if err != nil {
		return nil, err
	}

	// Get all local branches
	branches, err := g.ListBranches()
	if err != nil {
		return nil, err
	}

	// Get merged branches (via git)
	merged, err := g.MergedBranches(db)
	if err != nil {
		return nil, err
	}
	mergedMap := make(map[string]bool)
	for _, br := range merged {
		mergedMap[br] = true
	}

	// Get all PRs to check closed/merged status
	prs, err := ghc.ListPRs("all")
	if err != nil {
		// If GitHub API fails, continue with just git info
		prs = nil
	}

	// Build map of branches with closed/merged PRs
	closedPRBranches := make(map[string]bool)
	if prs != nil {
		for _, pr := range prs {
			if pr.State == "closed" || pr.Merged {
				closedPRBranches[pr.Head.Ref] = true
			}
		}
	}

	// Get list of remote branches
	remoteOut, err := g.Run("branch", "-r", "--format=%(refname:short)")
	remoteBranches := make(map[string]bool)
	if err == nil {
		for _, br := range strings.Split(strings.TrimSpace(remoteOut), "\n") {
			if br != "" && !strings.HasPrefix(br, "origin/HEAD") {
				// Strip "origin/" prefix
				br = strings.TrimPrefix(br, "origin/")
				remoteBranches[br] = true
			}
		}
	}

	// Collect branches to clean
	var localToDelete []string
	var remoteToDelete []string

	for _, br := range branches {
		// Skip current and default branches
		if br == db || br == cur || br == "" {
			continue
		}

		// Check if branch should be cleaned
		shouldClean := mergedMap[br] || closedPRBranches[br]

		if shouldClean {
			localToDelete = append(localToDelete, br)
			// Only add to remote delete list if it exists remotely
			if remoteBranches[br] {
				remoteToDelete = append(remoteToDelete, br)
			}
		}
	}

	return &CleanableBranches{
		LocalBranches:  localToDelete,
		RemoteBranches: remoteToDelete,
	}, nil
}

func DeleteLocalBranches(g git.Service, branches []string) []DeletionResult {
	results := make([]DeletionResult, len(branches))
	var wg sync.WaitGroup
	wg.Add(len(branches))
	for i, br := range branches {
		i, br := i, br
		go func() {
			defer wg.Done()
			err := g.DeleteBranch(br)
			results[i] = DeletionResult{Branch: br, Err: err}
		}()
	}
	wg.Wait()
	return results
}

func DeleteRemoteBranches(g git.Service, branches []string) []DeletionResult {
	results := make([]DeletionResult, len(branches))
	for i, br := range branches {
		err := g.DeleteRemoteBranch(br)
		if err != nil {
			// Check if the error is because the branch doesn't exist
			if strings.Contains(err.Error(), "remote ref does not exist") {
				// Branch is already deleted, not an error
				err = nil
			}
		}
		results[i] = DeletionResult{Branch: "origin/" + br, Err: err}
	}
	return results
}

// TODO: add support for deleting remote branches
