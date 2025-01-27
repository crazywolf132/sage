package cmd_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/cmd"
	"github.com/crazywolf132/sage/internal/gitutils"
)

func TestCommitCommand(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"add", "."}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"commit", "-m", "test commit"}).Return(nil).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
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
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"add", "."}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"commit", "-m", "test commit"}).Return(fmt.Errorf("nothing to commit, working tree clean")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nothing to commit")
		mockGit.AssertExpectations(t)
	})

	t.Run("too many arguments", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"commit", "test commit", "extra"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "accepts at most 1 arg(s)")
		mockGit.AssertExpectations(t)
	})

	t.Run("no commit message", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"add", "."}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"commit"}).Return(nil).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"commit"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
		mockGit.AssertExpectations(t)
	})

	t.Run("not a git repository", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(fmt.Errorf("fatal: not a git repository")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a git repository")
		mockGit.AssertExpectations(t)
	})

	t.Run("git add fails", func(t *testing.T) {
		// Create a fresh mock for this test
		mockGit := &gitutils.MockGitRunner{}
		gitutils.DefaultRunner = mockGit

		// Setup mock expectations for git repo check
		mockGit.On("RunGitCommand", []string{"rev-parse", "--git-dir"}).Return(nil).Once()
		mockGit.On("RunGitCommand", []string{"add", "."}).Return(fmt.Errorf("fatal: unable to add files")).Once()

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"commit", "test commit"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to add files")
		mockGit.AssertExpectations(t)
	})
}
