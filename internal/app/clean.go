package app

import (
	"sync"

	"github.com/crazywolf132/sage/internal/git"
)

type CleanableBranches struct {
	Branches []string
}

type DeletionResult struct {
	Branch string
	Err    error
}

func FindCleanableBranches(g git.Service) (*CleanableBranches, error) {
	isRepo, err := g.IsRepo()
	if err != nil || !isRepo {
		return nil, err
	}

	db, err := g.DefaultBranch()
	if err != nil {
		db = "main"
	}
	merged, err := g.MergedBranches(db)
	if err != nil {
		return nil, err
	}
	cur, err := g.CurrentBranch()
	if err != nil {
		return nil, err
	}

	var toDelete []string
	for _, br := range merged {
		if br == db || br == cur || br == "" {
			continue
		}
		toDelete = append(toDelete, br)
	}
	return &CleanableBranches{Branches: toDelete}, nil
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

// TODO: add support for deleting remote branches
