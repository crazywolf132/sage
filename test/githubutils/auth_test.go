package githubutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/crazywolf132/sage/internal/githubutils"
)

func TestGetGitHubToken(t *testing.T) {
	// Save original environment variables
	origSageToken := os.Getenv("SAGE_GITHUB_TOKEN")
	origGitHubToken := os.Getenv("GITHUB_TOKEN")
	origPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("SAGE_GITHUB_TOKEN", origSageToken)
		os.Setenv("GITHUB_TOKEN", origGitHubToken)
		os.Setenv("PATH", origPath)
	}()

	// Create a temporary directory for mock gh CLI
	tmpDir, err := os.MkdirTemp("", "mock-gh-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create mock gh CLI script
	mockGhPath := filepath.Join(tmpDir, "gh")
	err = os.WriteFile(mockGhPath, []byte(`#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "token" ]; then
  echo "mock-gh-token"
  exit 0
fi
exit 1
`), 0755)
	require.NoError(t, err)

	// Add temporary directory to PATH
	newPath := tmpDir + string(os.PathListSeparator) + origPath
	err = os.Setenv("PATH", newPath)
	require.NoError(t, err)

	t.Run("gh_cli_available", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("SAGE_GITHUB_TOKEN")
		os.Unsetenv("GITHUB_TOKEN")

		token, err := githubutils.GetGitHubToken()
		assert.NoError(t, err)
		assert.Equal(t, "mock-gh-token", token)
	})

	t.Run("gh_cli_error", func(t *testing.T) {
		// Create failing mock gh CLI script
		err := os.WriteFile(mockGhPath, []byte(`#!/bin/sh
exit 1
`), 0755)
		require.NoError(t, err)

		token, err := githubutils.GetGitHubToken()
		assert.NoError(t, err) // Error should be ignored
		assert.Empty(t, token)
	})

	// Remove gh from PATH for environment variable tests
	err = os.Setenv("PATH", "")
	require.NoError(t, err)

	t.Run("sage_token_env_var", func(t *testing.T) {
		os.Setenv("SAGE_GITHUB_TOKEN", "test-sage-token")
		os.Unsetenv("GITHUB_TOKEN")

		token, err := githubutils.GetGitHubToken()
		assert.NoError(t, err)
		assert.Equal(t, "test-sage-token", token)
	})

	t.Run("github_token_env_var", func(t *testing.T) {
		os.Unsetenv("SAGE_GITHUB_TOKEN")
		os.Setenv("GITHUB_TOKEN", "test-github-token")

		token, err := githubutils.GetGitHubToken()
		assert.NoError(t, err)
		assert.Equal(t, "test-github-token", token)
	})

	t.Run("sage_token_takes_precedence", func(t *testing.T) {
		os.Setenv("SAGE_GITHUB_TOKEN", "test-sage-token")
		os.Setenv("GITHUB_TOKEN", "test-github-token")

		token, err := githubutils.GetGitHubToken()
		assert.NoError(t, err)
		assert.Equal(t, "test-sage-token", token)
	})

	t.Run("no_token_available", func(t *testing.T) {
		os.Unsetenv("SAGE_GITHUB_TOKEN")
		os.Unsetenv("GITHUB_TOKEN")

		token, err := githubutils.GetGitHubToken()
		assert.NoError(t, err)
		assert.Empty(t, token)
	})
}
