package gitutils

import (
	"github.com/stretchr/testify/mock"
)

// GitRunner interface defines the contract for Git operations
type GitRunner interface {
	RunGitCommand(args ...string) error
	IsWorkingDirectoryClean() (bool, error)
	GetCurrentBranch() (string, error)
	IsMergeInProgress() (bool, error)
	IsRebaseInProgress() (bool, error)
}

// MockGitRunner is a mock implementation of GitRunner
type MockGitRunner struct {
	mock.Mock
}

// RunGitCommand is a mock implementation
func (m *MockGitRunner) RunGitCommand(args ...string) error {
	return m.Called(args).Error(0)
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
