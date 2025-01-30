package git

import (
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/crazywolf132/fstr"
	"github.com/sirupsen/logrus"
)

// CurrentBranch returns the name of the current branch.
func (r *Repo) CurrentBranch() (string, error) {
	branch, err := r.Git("symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", errors.Wrap(err, "failed to determine current branch")
	}
	return branch, nil
}

// DefaultBranch returns the name of the default branch for the remote.
func (r *Repo) DefaultBranchName() (string, error) {
	ref, err := r.Git("symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", errors.New("failed to determine remote HEAD")
	}
	return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
}

// BranchExists returns true if the branch exists.
func (r *Repo) BranchExists(name string, remote bool) (bool, error) {
	if remote {
		return r.DoesRefExist(fstr.F("refs/remotes/origin/{}", name))
	}
	return r.DoesRefExist(fstr.F("refs/heads/%s", name))
}

// IsTrunkBranch returns true if the branch is the default branch for the remote.
func (r *Repo) IsTrunkBranch(name string) (bool, error) {
	defaultBranch, err := r.DefaultBranchName()
	if err != nil {
		return false, err
	}
	return name == defaultBranch, nil
}

// IsCurrentBranchTrunk returns true if the current branch is the default branch for the remote.
func (r *Repo) IsCurrentBranchTrunk() (bool, error) {
	currentBranch, err := r.CurrentBranch()
	if err != nil {
		return false, err
	}
	return r.IsTrunkBranch(currentBranch)
}

// GetTrunkBranch returns the name of the trunk branch and ensures it exists
func (r *Repo) GetTrunkBranch() (string, error) {
	// Get the default branch name
	trunkName, err := r.DefaultBranchName()
	if err != nil {
		return "", fmt.Errorf("failed to determine trunk branch: %w", err)
	}

	// Verify the branch exists
	exists, err := r.BranchExists(trunkName, true)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", fmt.Errorf("trunk branch %q does not exist", trunkName)
	}

	return trunkName, nil
}

type SwitchOpts struct {
	// If true, create the branch if it doesn't exist.
	Create bool
	// Name of the new branch.
	Name string
	// Starting point for the new branch.
	NewHeadRef string
}

func (r *Repo) Switch(opts *SwitchOpts) (string, error) {
	previousBranch, er := r.CurrentBranch()
	if er != nil {
		return "", er
	}
	args := []string{"switch"}
	if opts.Create {
		args = append(args, "-c")
	}
	args = append(args, opts.Name)

	if opts.NewHeadRef != "" {
		args = append(args, opts.NewHeadRef)
	}

	result, err := r.Run(&RunOpts{
		Args: args,
	})
	if err != nil {
		return "", err
	}
	if result.ExitCode != 0 {
		logrus.WithFields(logrus.Fields{
			"stdout": string(result.Stdout),
			"stderr": string(result.Stderr),
		}).Debug("git switch failed")
		return "", errors.Errorf("failed to switch branch: %q: %s", opts.Name, string(result.Stderr))
	}
	return previousBranch, err
}

// ListBranches returns a list of all branches in the repository.
func (r *Repo) ListBranches() ([]string, error) {
	branches, err := r.Git("branch", "--list")
	if err != nil {
		return nil, err
	}

	var names []string
	for _, line := range strings.Split(branches, "\n") {
		if line == "" {
			continue
		}
		names = append(names, strings.TrimPrefix(line, "* "))
	}

	return names, nil
}

func (r *Repo) DefaultBranch() (string, error) {
	ref, err := r.Git("symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		logrus.WithError(err).Debug("failed to determine remote HEAD")
		return "", errors.New("failed to determine remote HEAD")
	}
	return strings.TrimPrefix(ref, "refs/remotes/origin/"), nil
}

// HasRemote returns true if the branch exists on remote.
func (r *Repo) HasRemote(name string) (bool, error) {
	return r.DoesRefExist(fstr.F("refs/remotes/origin/{}", name))
}

// Pull pulls the latest changes from the remote.
func (r *Repo) Pull() error {
	_, err := r.Run(&RunOpts{
		Args: []string{"pull"},
	})
	return err
}
