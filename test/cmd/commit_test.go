package cmd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/crazywolf132/sage/cmd"
	"github.com/crazywolf132/sage/internal/gitutils"
	"github.com/crazywolf132/sage/internal/ui"
)

// Original GetCommitDetails function
var originalGetCommitDetails = ui.GetCommitDetails

// Mock commit details for testing
func mockGetCommitDetails(useConventional bool) (ui.CommitForm, error) {
	return ui.CommitForm{
		Type:    "feat",
		Scope:   "",
		Message: "interactive commit message",
	}, nil
}

func TestCommitCommand(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil)
		mockGit.On("RunGitCommand", "add", ".").Return(nil)
		mockGit.On("RunGitCommand", "commit", "-m", "test commit").Return(nil)

		// Use the global root command
		rootCmd := cmd.RootCmd
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("clean working directory", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", "rev-parse", "--git-dir").Return(nil)
		mockGit.On("RunGitCommand", "add", ".").Return(nil)
		mockGit.On("RunGitCommand", "commit", "-m", "test commit").Return(fmt.Errorf("nothing to commit, working tree clean"))

		// Use the global root command
		rootCmd := cmd.RootCmd
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nothing to commit")
		mockGit.AssertExpectations(t)
	})

	t.Run("too many arguments", func(t *testing.T) {
		// Use the global root command
		rootCmd := cmd.RootCmd
		rootCmd.SetArgs([]string{"commit", "test commit", "extra arg"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts at most 1 arg(s)")
	})

	t.Run("no commit message", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Mock the UI interaction
		ui.GetCommitDetails = mockGetCommitDetails
		defer func() {
			ui.GetCommitDetails = originalGetCommitDetails
		}()

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", mock.Anything, mock.Anything).Return(nil).Times(2)
		mockGit.On("RunGitCommand", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		// Use the global root command
		rootCmd := cmd.RootCmd
		rootCmd.SetArgs([]string{"commit", "-c"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
		mockGit.AssertNumberOfCalls(t, "RunGitCommand", 3) // rev-parse, add, commit
	})

	t.Run("not a git repository", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", mock.Anything, mock.Anything).Return(fmt.Errorf("fatal: not a git repository"))

		// Use the global root command
		rootCmd := cmd.RootCmd
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
		mockGit.AssertNumberOfCalls(t, "RunGitCommand", 1) // only rev-parse
	})

	t.Run("git add fails", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", mock.Anything, mock.Anything).Return(nil).Once()
		mockGit.On("RunGitCommand", mock.Anything, mock.Anything).Return(fmt.Errorf("fatal: unable to add files"))

		// Use the global root command
		rootCmd := cmd.RootCmd
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to add files")
		mockGit.AssertNumberOfCalls(t, "RunGitCommand", 2) // rev-parse and add
	})
}
