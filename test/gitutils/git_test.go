package gitutils_test

import (
	"fmt"
	"testing"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/stretchr/testify/assert"
)

func setupMockGit(t *testing.T) *gitutils.MockGitRunner {
	mock := new(gitutils.MockGitRunner)
	return mock
}

func TestIsWorkingDirectoryClean(t *testing.T) {
	mock := setupMockGit(t)

	// Test clean directory
	mock.On("IsWorkingDirectoryClean").Return(true, nil).Once()
	clean, err := mock.IsWorkingDirectoryClean()
	assert.NoError(t, err)
	assert.True(t, clean)

	// Test dirty directory
	mock.On("IsWorkingDirectoryClean").Return(false, nil).Once()
	clean, err = mock.IsWorkingDirectoryClean()
	assert.NoError(t, err)
	assert.False(t, clean)

	mock.AssertExpectations(t)
}

func TestGetCurrentBranch(t *testing.T) {
	mock := setupMockGit(t)

	// Test successful branch retrieval
	mock.On("GetCurrentBranch").Return("main", nil).Once()
	branch, err := mock.GetCurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)

	mock.AssertExpectations(t)
}

func TestRunGitCommand(t *testing.T) {
	mock := setupMockGit(t)

	// Test successful command
	mock.On("RunGitCommand", []string{"status"}).Return(nil).Once()
	err := mock.RunGitCommand("status")
	assert.NoError(t, err)

	// Test failed command
	expectedErr := fmt.Errorf("git command failed")
	mock.On("RunGitCommand", []string{"checkout", "nonexistent-branch"}).Return(expectedErr).Once()
	err = mock.RunGitCommand("checkout", "nonexistent-branch")
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	mock.AssertExpectations(t)
}
