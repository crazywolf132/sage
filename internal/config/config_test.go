package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEnv holds the test environment configuration
type testEnv struct {
	tmpDir        string
	origWd        string
	origHome      string
	origAppData   string
	origXdgConfig string
}

func setupTestEnv(t *testing.T) *testEnv {
	env := &testEnv{}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "sage-config-test")
	require.NoError(t, err)
	env.tmpDir = tmpDir

	// Save original environment
	env.origWd, _ = os.Getwd()
	env.origHome = os.Getenv("HOME")
	env.origAppData = os.Getenv("APPDATA")
	env.origXdgConfig = os.Getenv("XDG_CONFIG_HOME")

	// Set up environment for test
	os.Chdir(tmpDir)
	os.Setenv("HOME", tmpDir)
	os.Setenv("APPDATA", tmpDir)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create directories
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, ".sage"), 0755))

	// Initialize Git repository
	require.NoError(t, initGitRepo(tmpDir))

	// Create empty config files to ensure they exist
	globalPath, err := globalPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(globalPath), 0755))
	writeEmptyTOML(t, globalPath)

	localPath := filepath.Join(tmpDir, ".sage", "config.toml")
	writeEmptyTOML(t, localPath)

	t.Logf("Test environment setup:")
	t.Logf("  Temp dir: %s", tmpDir)
	t.Logf("  Global config: %s", globalPath)
	t.Logf("  Local config: %s", localPath)

	return env
}

func initGitRepo(dir string) error {
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "commit", "--allow-empty", "-m", "Initial commit"},
	}

	for _, cmd := range cmds {
		c, err := git.SetupSecureCommand(cmd[0], cmd[1:]...)
		if err != nil {
			return err
		}
		c.Dir = dir
		if err := c.Run(); err != nil {
			return err
		}
	}

	return nil
}

func writeEmptyTOML(t *testing.T, path string) {
	data := map[string]string{}
	b, err := toml.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, b, 0644))
	t.Logf("Created empty TOML file: %s", path)
}

func (env *testEnv) cleanup() {
	// Restore original environment
	os.Chdir(env.origWd)
	os.Setenv("HOME", env.origHome)
	os.Setenv("APPDATA", env.origAppData)
	os.Setenv("XDG_CONFIG_HOME", env.origXdgConfig)

	// Clean up temporary directory
	os.RemoveAll(env.tmpDir)
}

func TestEncryptionDecryption(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "Simple string",
			value: "test-value",
		},
		{
			name:  "Empty string",
			value: "",
		},
		{
			name:  "Complex string",
			value: "Test123!@#$%^&*()",
		},
		{
			name:  "Long string",
			value: strings.Repeat("test", 100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := encryptValue(tt.value)
			require.NoError(t, err)
			assert.NotEqual(t, tt.value, encrypted, "encrypted value should not match original value")

			// Decrypt
			decrypted, err := decryptValue(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.value, decrypted, "decrypted value should match original")
		})
	}
}

func TestGlobalPath(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	t.Run("Windows path", func(t *testing.T) {
		if runtime.GOOS != "windows" {
			t.Skip("Skipping Windows test on non-Windows platform")
		}
		path, err := globalPath()
		require.NoError(t, err)
		expected := filepath.Join(env.tmpDir, "sage", "config.toml")
		assert.Equal(t, expected, path)
	})

	t.Run("Darwin path", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("Skipping Darwin test on non-Darwin platform")
		}
		path, err := globalPath()
		require.NoError(t, err)
		expected := filepath.Join(env.tmpDir, "Library", "Application Support", "sage", "config.toml")
		assert.Equal(t, expected, path)
	})

	t.Run("Linux with XDG_CONFIG_HOME", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			t.Skip("Skipping Linux test on non-Linux platform")
		}
		path, err := globalPath()
		require.NoError(t, err)
		expected := filepath.Join(env.tmpDir, "sage", "config.toml")
		assert.Equal(t, expected, path)
	})
}

func TestConfigOperations(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Reset global state
	globalData = map[string]string{}
	localData = map[string]string{}

	t.Run("Set and Get operations", func(t *testing.T) {
		// Test global config
		err := Set("test.key", "global-value", true)
		require.NoError(t, err)
		assert.Equal(t, "global-value", Get("test.key", false))

		// Test local config
		err = Set("test.key", "local-value", false)
		require.NoError(t, err)
		assert.Equal(t, "local-value", Get("test.key", true), "local value should be returned when useLocal is true")
		assert.Equal(t, "global-value", Get("test.key", false), "global value should be returned when useLocal is false")
	})

	t.Run("Sensitive key handling", func(t *testing.T) {
		// Try to set a sensitive key locally
		err := Set("api.api_key", "secret", false)
		assert.Error(t, err, "setting sensitive key locally should fail")

		// Set sensitive key globally
		err = Set("api.api_key", "secret", true)
		require.NoError(t, err)

		// Verify the value is encrypted
		raw := globalData["api.api_key"]
		assert.NotEqual(t, "secret", raw, "sensitive value should be encrypted")

		// Verify we can still get the decrypted value
		assert.Equal(t, "secret", Get("api.api_key", false))
	})

	t.Run("Config persistence", func(t *testing.T) {
		// Clear existing data
		globalData = map[string]string{}
		localData = map[string]string{}

		// Set some values
		require.NoError(t, Set("persist.test", "persist-value", true))
		require.NoError(t, Set("persist.local", "local-value", false))

		// Load configs fresh
		globalData = map[string]string{}
		localData = map[string]string{}
		require.NoError(t, LoadAllConfigs())

		// Verify values were loaded
		assert.Equal(t, "persist-value", Get("persist.test", false))
		assert.Equal(t, "local-value", Get("persist.local", true))
	})

	t.Run("Unset operation", func(t *testing.T) {
		// Set then unset global value
		require.NoError(t, Set("unset.test", "value", true))
		assert.Equal(t, "value", Get("unset.test", false))
		require.NoError(t, Unset("unset.test", true))
		assert.Empty(t, Get("unset.test", false))

		// Set then unset local value
		require.NoError(t, Set("unset.local", "value", false))
		assert.Equal(t, "value", Get("unset.local", true))
		require.NoError(t, Unset("unset.local", false))
		assert.Empty(t, Get("unset.local", true))
	})
}

func TestSensitiveKeyDetection(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	tests := []struct {
		name      string
		key       string
		sensitive bool
	}{
		{"API key", "api.api_key", true},
		{"GitHub token", "github.token", true},
		{"OpenAI key", "openai.api_key", true},
		{"AI key", "ai.api_key", true},
		{"Auth token", "auth.token", true},
		{"Credentials token", "credentials.token", true},
		{"Regular key", "regular.key", false},
		{"Case insensitive", "API.API_KEY", true},
		{"Mixed case", "Github.Token", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Set(tt.key, "test-value", false)
			if tt.sensitive {
				assert.Error(t, err, "expected error for sensitive key")
			} else {
				assert.NoError(t, err, "unexpected error for non-sensitive key")
			}
		})
	}
}

func TestExperimentalFeatures(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	t.Run("Feature enablement", func(t *testing.T) {
		// Test enabling a feature globally
		require.NoError(t, Set("experimental.rerere", "true", true))
		assert.True(t, IsExperimentalFeatureEnabled("rerere"))

		// Test enabling a feature locally
		require.NoError(t, Set("experimental.fsmonitor", "true", false))
		assert.True(t, IsExperimentalFeatureEnabled("fsmonitor"))

		// Test disabled feature
		assert.False(t, IsExperimentalFeatureEnabled("nonexistent"))
	})

	t.Run("Known features", func(t *testing.T) {
		features := GetExperimentalFeatures()
		assert.Contains(t, features, "rerere")
		assert.Contains(t, features, "commit-graph")
		assert.Contains(t, features, "fsmonitor")
		assert.Contains(t, features, "maintenance")
	})
}
