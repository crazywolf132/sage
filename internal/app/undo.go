package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
)

func Undo(g git.Service, count int) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repo")
	}

	merging, err := g.IsMerging()
	if err != nil {
		return err
	}
	if merging {
		return g.MergeAbort()
	}
	rebase, err := g.IsRebasing()
	if err != nil {
		return err
	}
	if rebase {
		return g.RebaseAbort()
	}

	// If count is not specified or is 1, do a single undo
	if count <= 1 {
		return g.ResetSoft("HEAD~1")
	}

	// For multiple commits, use HEAD~N where N is the count
	return g.ResetSoft(fmt.Sprintf("HEAD~%d", count))
}
