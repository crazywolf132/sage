package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
)

func PushCurrentBranch(g git.Service, force bool) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("no a repo or error checking repo: %v", err)
	}
	br, err := g.CurrentBranch()
	if err != nil {
		return err
	}
	return g.Push(br, force)
}
