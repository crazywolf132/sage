package git

import (
	"time"
)

type Service interface {
	IsRepo() (bool, error)
	IsClean() (bool, error)
	StageAll() error
	StageAllExcept(excludePaths []string) error
	IsPathStaged(path string) (bool, error)
	Commit(msg string, allowEmpty bool, stageAll bool) error
	CurrentBranch() (string, error)
	Push(branch string, force bool) error
	GetDiff() (string, error)
	DefaultBranch() (string, error)
	MergedBranches(base string) ([]string, error)
	DeleteBranch(name string) error
	FetchAll() error
	Checkout(name string) error
	Pull() error
	PullFF() error
	PullRebase() error
	CreateBranch(name string) error
	Merge(base string) error
	MergeAbort() error
	IsMerging() (bool, error)
	RebaseAbort() error
	IsRebasing() (bool, error)
	StatusPorcelain() (string, error)
	ResetSoft(ref string) error
	ListBranches() ([]string, error)
	Log(branch string, limit int, stats, all bool) (string, error)
	SquashCommits(startCommit string) error
	IsHeadBranch(branch string) (bool, error)
	GetFirstCommit() (string, error)
	RunInteractive(cmd string, args ...string) error
	GetBranchLastCommit(branch string) (time.Time, error)
	GetBranchCommitCount(branch string) (int, error)
	GetBranchMergeConflicts(branch string) (int, error)
	Stash(message string) error
	StashPop() error
	StashList() ([]string, error)
	GetMergeBase(branch1, branch2 string) (string, error)
	GetCommitCount(revisionRange string) (int, error)
	GetBranchDivergence(branch1, branch2 string) (int, error)
	GetCommitHash(ref string) (string, error)
	IsAncestor(commit1, commit2 string) (bool, error)
	SetConfig(key, value string, global bool) error
}

// SetConfig sets a git config value
func (g *ShellGit) SetConfig(key, value string, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	_, err := g.run(args...)
	return err
}
