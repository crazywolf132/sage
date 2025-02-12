package git

import (
	"fmt"
	"strings"
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
	hasChanges    bool              // Track if there are unstaged changes
	isRebasing    bool              // Track rebase state
	isMerging     bool              // Track merge state
	stagedFiles   map[string]bool   // Track staged files by path
	fileContents  map[string]string // Track file contents

	// Call tracking for tests
	calls map[string]int
}

// NewMockGit creates a new mock Git service
func NewMockGit() *MockGit {
	m := &MockGit{
		currentBranch: "main",
		isClean:       true,
		isRepo:        true,
		branches:      make(map[string]bool),
		commits:       make(map[string]string),
		staged:        make(map[string]bool),
		stashed:       make([]string, 0),
		stagedFiles:   make(map[string]bool),
		fileContents:  make(map[string]string),
		calls:         make(map[string]int),
	}
	// Initialize with main branch
	m.branches["main"] = true
	return m
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
	m.staged["mock-file"] = true
	m.hasChanges = true
	m.isClean = false
	return nil
}

// CurrentBranch implements Service.CurrentBranch
func (m *MockGit) CurrentBranch() (string, error) {
	m.trackCall("CurrentBranch")
	return m.currentBranch, nil
}

// Push implements Service.Push
func (m *MockGit) Push(branch string, forceType string) error {
	m.trackCall("Push")
	if err := validateRef(branch); err != nil {
		return err
	}
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
	if !allowEmpty && m.isClean && len(m.staged) == 0 {
		return fmt.Errorf("no changes to commit")
	}
	m.commits["mock-hash"] = msg
	m.staged = make(map[string]bool)
	m.hasChanges = false
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
	// Reset staged files
	m.stagedFiles = make(map[string]bool)

	// Add mock files for testing
	m.fileContents["include.txt"] = "include"
	m.fileContents["exclude.txt"] = "exclude"

	// Stage all files except excluded ones
	for path := range m.fileContents {
		excluded := false
		for _, excludePath := range excludePaths {
			if path == excludePath {
				excluded = true
				break
			}
		}
		if !excluded {
			m.stagedFiles[path] = true
		}
	}
	return nil
}

func (m *MockGit) IsPathStaged(path string) (bool, error) {
	m.trackCall("IsPathStaged")
	return m.stagedFiles[path], nil
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

// DeleteBranch implements Service.DeleteBranch
func (m *MockGit) DeleteBranch(name string) error {
	m.trackCall("DeleteBranch")
	if err := validateRef(name); err != nil {
		return err
	}
	if _, exists := m.branches[name]; !exists {
		return fmt.Errorf("branch %s does not exist", name)
	}
	if name == m.currentBranch {
		return fmt.Errorf("cannot delete the current branch")
	}
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
	m.isRebasing = true
	return fmt.Errorf("rebase conflicts")
}

func (m *MockGit) Merge(base string) error {
	m.trackCall("Merge")
	m.isMerging = true
	return fmt.Errorf("merge conflicts")
}

func (m *MockGit) MergeAbort() error {
	m.trackCall("MergeAbort")
	m.isMerging = false
	return nil
}

func (m *MockGit) IsMerging() (bool, error) {
	m.trackCall("IsMerging")
	return m.isMerging, nil
}

func (m *MockGit) RebaseAbort() error {
	m.trackCall("RebaseAbort")
	m.isRebasing = false
	return nil
}

func (m *MockGit) IsRebasing() (bool, error) {
	m.trackCall("IsRebasing")
	return m.isRebasing, nil
}

func (m *MockGit) StatusPorcelain() (string, error) {
	m.trackCall("StatusPorcelain")
	var status strings.Builder
	// Show staged files with "A" prefix
	for path := range m.stagedFiles {
		if m.stagedFiles[path] {
			status.WriteString(fmt.Sprintf("A  %s\n", path))
		}
	}
	// Show untracked files with "??" prefix
	for path := range m.fileContents {
		if !m.stagedFiles[path] {
			status.WriteString(fmt.Sprintf("?? %s\n", path))
		}
	}
	return status.String(), nil
}

func (m *MockGit) ResetSoft(ref string) error {
	m.trackCall("ResetSoft")
	m.isClean = false
	m.stagedFiles["reset.txt"] = true
	return nil
}

// ListBranches returns a list of all local branches
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
	// Return mock log with commit messages
	if stats {
		return "mock-hash\x00Test User\x001234567890\x00add file1\n1\t2\tfile1.txt\nmock-hash2\x00Test User\x001234567890\x00add file2\n1\t2\tfile2.txt", nil
	}
	return "mock-hash\x00Test User\x001234567890\x00test commit", nil
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
	if cmd == "rebase" && len(args) >= 2 && args[0] == "-i" {
		m.isRebasing = true
		return fmt.Errorf("rebase in progress")
	}
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

// GetBranchMergeConflicts implements Service.GetBranchMergeConflicts
func (m *MockGit) GetBranchMergeConflicts(branch string) (int, error) {
	m.trackCall("GetBranchMergeConflicts")
	if _, exists := m.branches[branch]; !exists {
		return 0, fmt.Errorf("branch %s does not exist", branch)
	}
	// Mock implementation returns 1 to simulate a conflict
	return 1, nil
}

func (m *MockGit) Stash(message string) error {
	m.trackCall("Stash")
	m.stashed = append(m.stashed, message)
	return nil
}

// StashPop implements Service.StashPop
func (m *MockGit) StashPop() error {
	m.trackCall("StashPop")
	if len(m.stashed) == 0 {
		return fmt.Errorf("no stash entries")
	}
	m.stashed = m.stashed[:len(m.stashed)-1]
	m.isClean = false
	m.hasChanges = true
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

// GetBranchDivergence implements Service.GetBranchDivergence
func (m *MockGit) GetBranchDivergence(branch1, branch2 string) (int, error) {
	m.trackCall("GetBranchDivergence")
	if _, exists := m.branches[branch1]; !exists {
		return 0, fmt.Errorf("branch %s does not exist", branch1)
	}
	if _, exists := m.branches[branch2]; !exists {
		return 0, fmt.Errorf("branch %s does not exist", branch2)
	}
	// Mock implementation returns 2 to simulate divergence
	return 2, nil
}

func (m *MockGit) GetCommitHash(ref string) (string, error) {
	m.trackCall("GetCommitHash")
	if ref == "HEAD" {
		return "mock-head-hash", nil
	}
	return "mock-commit-hash", nil
}

func (m *MockGit) IsAncestor(commit1, commit2 string) (bool, error) {
	m.trackCall("IsAncestor")
	// Mock implementation: always return true for test commits
	if strings.HasPrefix(commit1, "mock-") && strings.HasPrefix(commit2, "mock-") {
		return true, nil
	}
	return false, nil
}
