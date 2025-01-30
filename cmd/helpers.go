package cmd

import (
	"os/exec"
	"strings"

	"emperror.dev/errors"
	"github.com/crazywolf132/sage/internal/git"
)

var cachedRepo *git.Repo

func getRepo() (*git.Repo, error) {
	if cachedRepo == nil {
		cmd := exec.Command(
			"git",
			"rev-parse",
			"--path-format=absolute",
			"--show-toplevel",
			"--git-common-dir",
		)

		paths, err := cmd.Output()
		if err != nil {
			return nil, errors.Wrap(
				err,
				"failed to find git directory (are you running inside a Git repo?)",
			)
		}

		dir, gitDir, found := strings.Cut(strings.TrimSpace(string(paths)), "\n")
		if !found {
			return nil, errors.New("Unexpected format, not able to parse toplevel and common dir.")
		}

		cachedRepo, err = git.OpenRepo(dir, gitDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to open git repo")
		}
	}
	return cachedRepo, nil
}
