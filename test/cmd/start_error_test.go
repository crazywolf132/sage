package cmd_test

import (
	"fmt"
	"testing"

	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/stretchr/testify/assert"
)

func TestStartCommandWithGitErrors(t *testing.T) {
	mockGit := new(gitutils.MockGitRunner)

	// Test error when getting current branch
	expectedErr := fmt.Errorf("failed to get current branch")
	mockGit.On("GetCurrentBranch").Return("", expectedErr).Once()
	branch, err := mockGit.GetCurrentBranch()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Empty(t, branch)

	// Test error when checking working directory status
	expectedErr = fmt.Errorf("failed to check working directory status")
	mockGit.On("IsWorkingDirectoryClean").Return(false, expectedErr).Once()
	clean, err := mockGit.IsWorkingDirectoryClean()
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.False(t, clean)

	// Test error when creating new branch
	expectedErr = fmt.Errorf("failed to create branch")
	mockGit.On("RunGitCommand", "switch", "-c", "feature/test").Return(expectedErr).Once()
	err = mockGit.RunGitCommand("switch", "-c", "feature/test")
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	// Test error when pushing to remote
	expectedErr = fmt.Errorf("failed to push to remote")
	mockGit.On("RunGitCommand", "push", "-u", "origin", "feature/test").Return(expectedErr).Once()
	err = mockGit.RunGitCommand("push", "-u", "origin", "feature/test")
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)

	mockGit.AssertExpectations(t)
}
