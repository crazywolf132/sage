package git

import (
	"fmt"
	"time"
)

// MockGit implements the Service interface for testing
type MockGit struct {
	// State
	currentBranch string
	isClean       bool
	isRepo        bool
	branches      map[string]bool
	commits       map[string]string // hash -> message
	staged        map[string]bool
	stashed       []string

	// Call tracking for tests
	calls map[string]int
}

// NewMockGit creates a new mock Git service
func NewMockGit() *MockGit {
	return &MockGit{
		currentBranch: "main",
		isClean:       true,
		isRepo:        true,
		branches:      make(map[string]bool),
		commits:       make(map[string]string),
		staged:        make(map[string]bool),
		stashed:       make([]string, 0),
		calls:         make(map[string]int),
	}
}

func (m *MockGit) trackCall(method string) {
	m.calls[method]++
}

// IsRepo implements Service.IsRepo
func (m *MockGit) IsRepo() (bool, error) {
	m.trackCall("IsRepo")
	return m.isRepo, nil
}

// IsClean implements Service.IsClean
func (m *MockGit) IsClean() (bool, error) {
	m.trackCall("IsClean")
	return m.isClean, nil
}

// StageAll implements Service.StageAll
func (m *MockGit) StageAll() error {
	m.trackCall("StageAll")
	return nil
}

// CurrentBranch implements Service.CurrentBranch
func (m *MockGit) CurrentBranch() (string, error) {
	m.trackCall("CurrentBranch")
	return m.currentBranch, nil
}

// Push implements Service.Push
func (m *MockGit) Push(branch string, force bool) error {
	m.trackCall("Push")
	if _, exists := m.branches[branch]; !exists {
		return fmt.Errorf("branch %s does not exist", branch)
	}
	return nil
}

// PushWithLease implements Service.PushWithLease
func (m *MockGit) PushWithLease(branch string) error {
	m.trackCall("PushWithLease")
	if _, exists := m.branches[branch]; !exists {
		return fmt.Errorf("branch %s does not exist", branch)
	}
	return nil
}

// CreateBranch implements Service.CreateBranch
func (m *MockGit) CreateBranch(name string) error {
	m.trackCall("CreateBranch")
	if err := validateRef(name); err != nil {
		return err
	}
	m.branches[name] = true
	return nil
}

// Checkout implements Service.Checkout
func (m *MockGit) Checkout(name string) error {
	m.trackCall("Checkout")
	if err := validateRef(name); err != nil {
		return err
	}
	if _, exists := m.branches[name]; !exists {
		return fmt.Errorf("branch %s does not exist", name)
	}
	m.currentBranch = name
	return nil
}

// Commit implements Service.Commit
func (m *MockGit) Commit(msg string, allowEmpty bool, stageAll bool) error {
	m.trackCall("Commit")
	if !allowEmpty && len(m.staged) == 0 {
		return fmt.Errorf("no changes to commit")
	}
	m.commits["mock-hash"] = msg
	m.staged = make(map[string]bool)
	m.isClean = true
	return nil
}

// GetCallCount returns the number of times a method was called
func (m *MockGit) GetCallCount(method string) int {
	return m.calls[method]
}

// SetClean sets the clean state for testing
func (m *MockGit) SetClean(clean bool) {
	m.isClean = clean
}

// SetCurrentBranch sets the current branch for testing
func (m *MockGit) SetCurrentBranch(branch string) {
	m.currentBranch = branch
	m.branches[branch] = true
}

// AddBranch adds a branch for testing
func (m *MockGit) AddBranch(name string) {
	m.branches[name] = true
}

// The following methods implement the rest of the Service interface with minimal functionality

func (m *MockGit) StageAllExcept(excludePaths []string) error {
	m.trackCall("StageAllExcept")
	return nil
}

func (m *MockGit) IsPathStaged(path string) (bool, error) {
	m.trackCall("IsPathStaged")
	return m.staged[path], nil
}

func (m *MockGit) GetDiff() (string, error) {
	m.trackCall("GetDiff")
	return "", nil
}

func (m *MockGit) DefaultBranch() (string, error) {
	m.trackCall("DefaultBranch")
	return "main", nil
}

func (m *MockGit) MergedBranches(base string) ([]string, error) {
	m.trackCall("MergedBranches")
	return []string{}, nil
}

func (m *MockGit) DeleteBranch(name string) error {
	m.trackCall("DeleteBranch")
	delete(m.branches, name)
	return nil
}

func (m *MockGit) FetchAll() error {
	m.trackCall("FetchAll")
	return nil
}

func (m *MockGit) Pull() error {
	m.trackCall("Pull")
	return nil
}

func (m *MockGit) PullFF() error {
	m.trackCall("PullFF")
	return nil
}

func (m *MockGit) PullRebase() error {
	m.trackCall("PullRebase")
	return nil
}

func (m *MockGit) PullMerge() error {
	m.trackCall("PullMerge")
	return nil
}

func (m *MockGit) Merge(base string) error {
	m.trackCall("Merge")
	return nil
}

func (m *MockGit) MergeAbort() error {
	m.trackCall("MergeAbort")
	return nil
}

func (m *MockGit) IsMerging() (bool, error) {
	m.trackCall("IsMerging")
	return false, nil
}

func (m *MockGit) RebaseAbort() error {
	m.trackCall("RebaseAbort")
	return nil
}

func (m *MockGit) IsRebasing() (bool, error) {
	m.trackCall("IsRebasing")
	return false, nil
}

func (m *MockGit) StatusPorcelain() (string, error) {
	m.trackCall("StatusPorcelain")
	return "", nil
}

func (m *MockGit) ResetSoft(ref string) error {
	m.trackCall("ResetSoft")
	return nil
}

func (m *MockGit) ListBranches() ([]string, error) {
	m.trackCall("ListBranches")
	branches := make([]string, 0, len(m.branches))
	for b := range m.branches {
		branches = append(branches, b)
	}
	return branches, nil
}

func (m *MockGit) Log(branch string, limit int, stats, all bool) (string, error) {
	m.trackCall("Log")
	return "", nil
}

func (m *MockGit) SquashCommits(startCommit string) error {
	m.trackCall("SquashCommits")
	return nil
}

func (m *MockGit) IsHeadBranch(branch string) (bool, error) {
	m.trackCall("IsHeadBranch")
	return m.currentBranch == branch, nil
}

func (m *MockGit) GetFirstCommit() (string, error) {
	m.trackCall("GetFirstCommit")
	return "mock-first-commit", nil
}

func (m *MockGit) RunInteractive(cmd string, args ...string) error {
	m.trackCall("RunInteractive")
	return nil
}

func (m *MockGit) GetBranchLastCommit(branch string) (time.Time, error) {
	m.trackCall("GetBranchLastCommit")
	return time.Now(), nil
}

func (m *MockGit) GetBranchCommitCount(branch string) (int, error) {
	m.trackCall("GetBranchCommitCount")
	return 1, nil
}

func (m *MockGit) GetBranchMergeConflicts(branch string) (int, error) {
	m.trackCall("GetBranchMergeConflicts")
	return 0, nil
}

func (m *MockGit) Stash(message string) error {
	m.trackCall("Stash")
	m.stashed = append(m.stashed, message)
	return nil
}

func (m *MockGit) StashPop() error {
	m.trackCall("StashPop")
	if len(m.stashed) == 0 {
		return fmt.Errorf("no stash entries")
	}
	m.stashed = m.stashed[:len(m.stashed)-1]
	return nil
}

func (m *MockGit) StashList() ([]string, error) {
	m.trackCall("StashList")
	return m.stashed, nil
}

func (m *MockGit) GetMergeBase(branch1, branch2 string) (string, error) {
	m.trackCall("GetMergeBase")
	return "mock-merge-base", nil
}

func (m *MockGit) GetCommitCount(revisionRange string) (int, error) {
	m.trackCall("GetCommitCount")
	return 1, nil
}

func (m *MockGit) GetBranchDivergence(branch1, branch2 string) (int, error) {
	m.trackCall("GetBranchDivergence")
	return 0, nil
}

func (m *MockGit) GetCommitHash(ref string) (string, error) {
	m.trackCall("GetCommitHash")
	return "mock-commit-hash", nil
}

func (m *MockGit) IsAncestor(commit1, commit2 string) (bool, error) {
	m.trackCall("IsAncestor")
	return true, nil
}

// DeleteRemoteBranch implements Service.DeleteRemoteBranch
func (m *MockGit) DeleteRemoteBranch(name string) error {
	m.trackCall("DeleteRemoteBranch")
	if err := validateRef(name); err != nil {
		return err
	}
	delete(m.branches, "origin/"+name)
	return nil
}

// StagedDiff returns the diff of staged changes
func (m *MockGit) StagedDiff() (string, error) {
	m.trackCall("StagedDiff")
	return "", nil
}

// GrepDiff searches for a pattern in a diff and returns matching lines
func (m *MockGit) GrepDiff(diff string, pattern string) ([]string, error) {
	m.trackCall("GrepDiff")
	return []string{}, nil
}

// ListConflictedFiles returns a list of files with conflicts
func (m *MockGit) ListConflictedFiles() (string, error) {
	m.trackCall("ListConflictedFiles")
	return "", nil
}

// GetConfigValue returns a mock Git configuration value
func (m *MockGit) GetConfigValue(key string) (string, error) {
	m.trackCall("GetConfigValue")
	return "", nil
}

// MergeContinue continues a merge operation
func (m *MockGit) MergeContinue() error {
	m.trackCall("MergeContinue")
	return nil
}

// RebaseContinue continues a rebase operation
func (m *MockGit) RebaseContinue() error {
	m.trackCall("RebaseContinue")
	return nil
}
