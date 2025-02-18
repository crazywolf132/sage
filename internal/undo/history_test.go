package undo

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGit is a mock implementation of git.Service
type MockGit struct {
	mock.Mock
}

func (m *MockGit) Run(args ...string) (string, error) {
	arguments := make([]interface{}, len(args))
	for i, v := range args {
		arguments[i] = v
	}
	result := m.Called(arguments...)
	return result.String(0), result.Error(1)
}

// Implement other required methods from git.Service interface with empty implementations
func (m *MockGit) IsRepo() (bool, error)                                         { return true, nil }
func (m *MockGit) IsClean() (bool, error)                                        { return true, nil }
func (m *MockGit) StageAll() error                                               { return nil }
func (m *MockGit) StageAllExcept(excludePaths []string) error                    { return nil }
func (m *MockGit) IsPathStaged(path string) (bool, error)                        { return true, nil }
func (m *MockGit) Commit(msg string, allowEmpty bool, stageAll bool) error       { return nil }
func (m *MockGit) CurrentBranch() (string, error)                                { return "", nil }
func (m *MockGit) Push(branch string, force bool) error                          { return nil }
func (m *MockGit) PushWithLease(branch string) error                             { return nil }
func (m *MockGit) GetDiff() (string, error)                                      { return "", nil }
func (m *MockGit) DefaultBranch() (string, error)                                { return "", nil }
func (m *MockGit) MergedBranches(base string) ([]string, error)                  { return nil, nil }
func (m *MockGit) DeleteBranch(name string) error                                { return nil }
func (m *MockGit) DeleteRemoteBranch(name string) error                          { return nil }
func (m *MockGit) FetchAll() error                                               { return nil }
func (m *MockGit) Checkout(name string) error                                    { return nil }
func (m *MockGit) Pull() error                                                   { return nil }
func (m *MockGit) PullFF() error                                                 { return nil }
func (m *MockGit) PullRebase() error                                             { return nil }
func (m *MockGit) CreateBranch(name string) error                                { return nil }
func (m *MockGit) Merge(base string) error                                       { return nil }
func (m *MockGit) MergeAbort() error                                             { return nil }
func (m *MockGit) IsMerging() (bool, error)                                      { return false, nil }
func (m *MockGit) RebaseAbort() error                                            { return nil }
func (m *MockGit) IsRebasing() (bool, error)                                     { return false, nil }
func (m *MockGit) StatusPorcelain() (string, error)                              { return "", nil }
func (m *MockGit) ResetSoft(ref string) error                                    { return nil }
func (m *MockGit) ListBranches() ([]string, error)                               { return nil, nil }
func (m *MockGit) Log(branch string, limit int, stats, all bool) (string, error) { return "", nil }
func (m *MockGit) SquashCommits(startCommit string) error                        { return nil }
func (m *MockGit) IsHeadBranch(branch string) (bool, error)                      { return false, nil }
func (m *MockGit) GetFirstCommit() (string, error)                               { return "", nil }
func (m *MockGit) RunInteractive(cmd string, args ...string) error               { return nil }
func (m *MockGit) GetBranchLastCommit(branch string) (time.Time, error)          { return time.Time{}, nil }
func (m *MockGit) GetBranchCommitCount(branch string) (int, error)               { return 0, nil }
func (m *MockGit) GetBranchMergeConflicts(branch string) (int, error)            { return 0, nil }
func (m *MockGit) Stash(message string) error                                    { return nil }
func (m *MockGit) StashPop() error                                               { return nil }
func (m *MockGit) StashList() ([]string, error)                                  { return nil, nil }
func (m *MockGit) GetMergeBase(branch1, branch2 string) (string, error)          { return "", nil }
func (m *MockGit) GetCommitCount(revisionRange string) (int, error)              { return 0, nil }
func (m *MockGit) GetBranchDivergence(branch1, branch2 string) (int, error)      { return 0, nil }
func (m *MockGit) GetCommitHash(ref string) (string, error)                      { return "", nil }
func (m *MockGit) IsAncestor(commit1, commit2 string) (bool, error)              { return false, nil }
func (m *MockGit) SetConfig(key, value string, global bool) error                { return nil }
func (m *MockGit) GetRepoPath() (string, error)                                  { return "", nil }

func TestNewHistory(t *testing.T) {
	h := NewHistory()
	assert.NotNil(t, h)
	assert.Equal(t, 100, h.MaxSize)
	assert.Empty(t, h.Operations)
}

func TestAddOperation(t *testing.T) {
	h := NewHistory()
	op := Operation{
		ID:          "test-id",
		Type:        "commit",
		Description: "Test commit",
		Command:     "git commit -m 'test'",
		Timestamp:   time.Now(),
		Ref:         "abc123",
		Category:    "commit",
	}

	h.AddOperation(op)
	assert.Len(t, h.Operations, 1)
	assert.Equal(t, op, h.Operations[0])

	// Test max size limit
	h.MaxSize = 2
	h.AddOperation(Operation{ID: "test-id-2"})
	h.AddOperation(Operation{ID: "test-id-3"})
	assert.Len(t, h.Operations, 2)
	assert.Equal(t, "test-id-3", h.Operations[0].ID)
	assert.Equal(t, "test-id-2", h.Operations[1].ID)
}

func TestGetOperations(t *testing.T) {
	h := NewHistory()
	now := time.Now()

	ops := []Operation{
		{ID: "1", Category: "commit", Timestamp: now.Add(-2 * time.Hour)},
		{ID: "2", Category: "merge", Timestamp: now.Add(-1 * time.Hour)},
		{ID: "3", Category: "commit", Timestamp: now},
	}

	for _, op := range ops {
		h.AddOperation(op)
	}

	// Test filtering by category
	commits := h.GetOperations("commit", time.Time{})
	assert.Len(t, commits, 2)
	assert.Equal(t, "3", commits[0].ID)
	assert.Equal(t, "1", commits[1].ID)

	// Test filtering by time
	recent := h.GetOperations("", now.Add(-30*time.Minute))
	assert.Len(t, recent, 1)
	assert.Equal(t, "3", recent[0].ID)
}

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sage-test-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a mock git service and set up test directory structure
	mockGit := &MockGit{}
	gitDir := filepath.Join(tmpDir, ".git")
	sageDir := filepath.Join(gitDir, ".sage")
	err = os.MkdirAll(sageDir, 0755)
	assert.NoError(t, err)

	// Setup mock expectations for both Save and Load operations
	mockGit.On("Run", []interface{}{"rev-parse", "--git-dir"}...).Return(gitDir, nil).Twice()

	h := NewHistory().WithGitService(mockGit)
	op := Operation{
		ID:          "test-id",
		Type:        "commit",
		Description: "Test commit",
		Command:     "git commit -m 'test'",
		Timestamp:   time.Now(),
		Ref:         "abc123",
		Category:    "commit",
	}
	h.AddOperation(op)

	// Test Save
	err = h.Save(tmpDir)
	assert.NoError(t, err)

	// Verify file was created
	historyPath := filepath.Join(sageDir, "undo_history.json")
	_, err = os.Stat(historyPath)
	assert.NoError(t, err)

	// Test Load with a new history instance
	h2 := NewHistory().WithGitService(mockGit)
	err = h2.Load(tmpDir)
	assert.NoError(t, err)

	// Compare operations, ignoring timestamps
	assert.Equal(t, len(h.Operations), len(h2.Operations))
	for i := range h.Operations {
		h.Operations[i].Timestamp = time.Time{}
		h2.Operations[i].Timestamp = time.Time{}
	}
	assert.Equal(t, h.Operations, h2.Operations)

	// Verify all mock expectations were met
	mockGit.AssertExpectations(t)
}

func TestClear(t *testing.T) {
	h := NewHistory()
	h.AddOperation(Operation{ID: "1"})
	h.AddOperation(Operation{ID: "2"})

	h.Clear()
	assert.Empty(t, h.Operations)
}
