package git

import (
	"emperror.dev/errors"
	"github.com/crazywolf132/fstr"
	"github.com/go-git/go-git/v5"
)

const DEFAULT_REMOTE_NAME = "origin"

type Repo struct {
	repoDir string
	gitDir  string
	gitRepo *git.Repository
}

func OpenRepo(repoDir, gitDir string) (*Repo, error) {
	repo, err := git.PlainOpenWithOptions(repoDir, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, errors.Errorf("failed to open git repo: %v", err)
	}
	r := &Repo{
		repoDir,
		gitDir,
		repo,
	}
	return r, nil
}

// GetRemoteName returns the name of the remote for the current branch.
func (r *Repo) GetRemoteName() string {
	return DEFAULT_REMOTE_NAME
}

// CurrentBranchRemote returns the name of the remote for the current branch.
func (r *Repo) CurrentBranchRemote() (string, error) {
	branch, err := r.CurrentBranch()
	if err != nil {
		return "", err
	}
	remote, err := r.Git("config", "--get", fstr.F("branch.{}.remote", branch))
	if err != nil {
		return "", err
	}
	return remote, nil
}
