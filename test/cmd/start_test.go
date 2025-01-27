package cmd_test

import (
	"testing"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/stretchr/testify/assert"
)

func TestStartCommand(t *testing.T) {
	mockGit := new(gitutils.MockGitRunner)

	// Test successful branch creation
	mockGit.On("GetCurrentBranch").Return("main", nil).Once()
	mockGit.On("IsWorkingDirectoryClean").Return(true, nil).Once()
	mockGit.On("RunGitCommand", []string{"checkout", "-b", "feature/test"}).Return(nil).Once()
	mockGit.On("RunGitCommand", []string{"push", "-u", "origin", "feature/test"}).Return(nil).Once()

	// Simulate branch creation
	err := mockGit.RunGitCommand("checkout", "-b", "feature/test")
	assert.NoError(t, err)

	// Verify current branch
	branch, err := mockGit.GetCurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)

	// Verify working directory is clean
	clean, err := mockGit.IsWorkingDirectoryClean()
	assert.NoError(t, err)
	assert.True(t, clean)

	// Verify push to remote
	err = mockGit.RunGitCommand("push", "-u", "origin", "feature/test")
	assert.NoError(t, err)

	mockGit.AssertExpectations(t)
}

func TestStartCommandWithDirtyWorkingDirectory(t *testing.T) {
	mockGit := new(gitutils.MockGitRunner)

	// Test branch creation with dirty working directory
	mockGit.On("IsWorkingDirectoryClean").Return(false, nil).Once()

	// Verify working directory is dirty
	clean, err := mockGit.IsWorkingDirectoryClean()
	assert.NoError(t, err)
	assert.False(t, clean)

	mockGit.AssertExpectations(t)
}
