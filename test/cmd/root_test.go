package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/crazywolf132/sage/cmd"
)

func TestRootCommand(t *testing.T) {
	t.Run("custom config file", func(t *testing.T) {
		// Create a temporary config file
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "custom.yaml")
		err := os.WriteFile(configFile, []byte("defaultBranch: main"), 0644)
		require.NoError(t, err)

		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"--config", configFile})
		err = rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
	})

	t.Run("invalid config file", func(t *testing.T) {
		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"--config", "nonexistent.yaml"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err) // Config errors are not fatal
	})

	t.Run("explain flag", func(t *testing.T) {
		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"--explain"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
	})

	t.Run("help command", func(t *testing.T) {
		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"--help"})
		err := rootCmd.Execute()

		// Assert
		assert.NoError(t, err)
	})

	t.Run("unknown command", func(t *testing.T) {
		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"unknown"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown command")
	})

	t.Run("unknown flag", func(t *testing.T) {
		// Create a fresh command instance
		rootCmd := cmd.NewRootCmd()
		rootCmd.SetArgs([]string{"--unknown"})
		err := rootCmd.Execute()

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown flag")
	})
}
