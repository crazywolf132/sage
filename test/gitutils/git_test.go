package gitutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/gitutils"
)

func TestIsWorkingDirectoryClean(t *testing.T) {
	mockGit := &gitutils.MockGitRunner{}
	gitutils.DefaultRunner = mockGit

	mockGit.On("IsWorkingDirectoryClean").Return(true, nil).Once()

	clean, err := gitutils.DefaultRunner.IsWorkingDirectoryClean()
	assert.NoError(t, err)
	assert.True(t, clean)
	mockGit.AssertExpectations(t)
}

func TestGetCurrentBranch(t *testing.T) {
	mockGit := &gitutils.MockGitRunner{}
	gitutils.DefaultRunner = mockGit

	mockGit.On("GetCurrentBranch").Return("main", nil).Once()

	branch, err := gitutils.DefaultRunner.GetCurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)
	mockGit.AssertExpectations(t)
}

func TestRunGitCommand(t *testing.T) {
	mockGit := &gitutils.MockGitRunner{}
	gitutils.DefaultRunner = mockGit

	args := []interface{}{"status"}
	mockGit.On("RunGitCommand", args...).Return(nil).Once()

	err := gitutils.DefaultRunner.RunGitCommand("status")
	assert.NoError(t, err)
	mockGit.AssertExpectations(t)
}
