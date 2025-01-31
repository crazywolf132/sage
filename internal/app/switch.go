package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
)

func SwitchBranch(g git.Service, branch string) error {
	repo, err := g.IsRepo()
	if err != nil || !repo {
		return fmt.Errorf("not a git repo")
	}
	return g.Checkout(branch)
}
