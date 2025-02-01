package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
)

func Undo(g git.Service) error {
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
	// do a soft reset HEAD~1, or fail
	return g.ResetSoft("HEAD~1")
}
