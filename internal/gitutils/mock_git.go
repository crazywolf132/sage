package gitutils

import (
	"github.com/stretchr/testify/mock"
)

// GitRunner interface defines the contract for Git operations
type GitRunner interface {
	RunGitCommand(args ...string) error
	RunGitCommandWithOutput(args ...string) (string, error)
	IsWorkingDirectoryClean() (bool, error)
	GetCurrentBranch() (string, error)
	IsMergeInProgress() (bool, error)
	IsRebaseInProgress() (bool, error)
	GetBranches() ([]string, error)
	BranchExists(branchName string) (bool, error)
	GetFirstCommitOnBranch() (string, error)
}

// MockGitRunner is a mock implementation of GitRunner
type MockGitRunner struct {
	mock.Mock
}

// RunGitCommand is a mock implementation
func (m *MockGitRunner) RunGitCommand(args ...string) error {
	// Convert []string to []interface{}
	iargs := make([]interface{}, len(args))
	for i, v := range args {
		iargs[i] = v
	}
	return m.Called(iargs...).Error(0)
}

// IsWorkingDirectoryClean is a mock implementation
func (m *MockGitRunner) IsWorkingDirectoryClean() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

// GetCurrentBranch is a mock implementation
func (m *MockGitRunner) GetCurrentBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

// IsMergeInProgress is a mock implementation
func (m *MockGitRunner) IsMergeInProgress() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

// IsRebaseInProgress is a mock implementation
func (m *MockGitRunner) IsRebaseInProgress() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

// GetBranches is a mock implementation
func (m *MockGitRunner) GetBranches() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// BranchExists is a mock implementation
func (m *MockGitRunner) BranchExists(branchName string) (bool, error) {
	args := m.Called(branchName)
	return args.Bool(0), args.Error(1)
}

// RunGitCommandWithOutput is a mock implementation
func (m *MockGitRunner) RunGitCommandWithOutput(args ...string) (string, error) {
	// Convert []string to []interface{}
	iargs := make([]interface{}, len(args))
	for i, v := range args {
		iargs[i] = v
	}
	args2 := m.Called(iargs...)
	return args2.String(0), args2.Error(1)
}

// GetFirstCommitOnBranch is a mock implementation
func (m *MockGitRunner) GetFirstCommitOnBranch() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}
