package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
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
	if err != nil {
		t.Fatal(err)
	}
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
	os.MkdirAll(filepath.Join(tmpDir, ".sage"), 0755)

	// Initialize Git repository
	if err := initGitRepo(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create empty config files to ensure they exist
	globalPath, err := globalPath()
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(globalPath), 0755)
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
		c := exec.Command(cmd[0], cmd[1:]...)
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
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("Created empty TOML file: %s", path)
}

func dumpConfig(t *testing.T, msg string) {
	t.Logf("=== Config Dump: %s ===", msg)
	t.Logf("Global data: %v", globalData)
	t.Logf("Local data: %v", localData)

	// Read actual file contents
	if gp, err := globalPath(); err == nil {
		if data, err := os.ReadFile(gp); err == nil {
			t.Logf("Global file contents: %s", string(data))
		}
	}
	if data, err := os.ReadFile(".sage/config.toml"); err == nil {
		t.Logf("Local file contents: %s", string(data))
	}
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
			if err != nil {
				t.Fatalf("encryptValue() error = %v", err)
			}
			if encrypted == tt.value {
				t.Error("encrypted value should not match original value")
			}

			// Decrypt
			decrypted, err := decryptValue(encrypted)
			if err != nil {
				t.Fatalf("decryptValue() error = %v", err)
			}
			if decrypted != tt.value {
				t.Errorf("decrypted value = %v, want %v", decrypted, tt.value)
			}
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
		if err != nil {
			t.Fatalf("globalPath() error = %v", err)
		}
		expected := filepath.Join(env.tmpDir, "sage", "config.toml")
		if path != expected {
			t.Errorf("globalPath() = %v, want %v", path, expected)
		}
	})

	t.Run("Darwin path", func(t *testing.T) {
		if runtime.GOOS != "darwin" {
			t.Skip("Skipping Darwin test on non-Darwin platform")
		}
		path, err := globalPath()
		if err != nil {
			t.Fatalf("globalPath() error = %v", err)
		}
		expected := filepath.Join(env.tmpDir, "Library", "Application Support", "sage", "config.toml")
		if path != expected {
			t.Errorf("globalPath() = %v, want %v", path, expected)
		}
	})

	t.Run("Linux with XDG_CONFIG_HOME", func(t *testing.T) {
		if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
			t.Skip("Skipping Linux test on non-Linux platform")
		}
		path, err := globalPath()
		if err != nil {
			t.Fatalf("globalPath() error = %v", err)
		}
		expected := filepath.Join(env.tmpDir, "sage", "config.toml")
		if path != expected {
			t.Errorf("globalPath() = %v, want %v", path, expected)
		}
	})
}

func TestConfigOperations(t *testing.T) {
	env := setupTestEnv(t)
	defer env.cleanup()

	// Test Set and Get operations
	t.Run("Set and Get", func(t *testing.T) {
		// Clear any existing data
		globalData = map[string]string{}
		localData = map[string]string{}

		dumpConfig(t, "Initial state")

		// Test global config
		err := Set("test.key", "global-value", true)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		dumpConfig(t, "After setting global value")

		if got := Get("test.key", false); got != "global-value" {
			t.Errorf("Get() with useLocal=false = %v, want %v", got, "global-value")
		}

		// Test local config
		err = Set("test.key", "local-value", false)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}
		dumpConfig(t, "After setting local value")

		// Local value should be returned when useLocal is true
		if got := Get("test.key", true); got != "local-value" {
			t.Errorf("Get() with useLocal=true = %v, want %v", got, "local-value")
		}

		// Global value should be returned when useLocal is false
		if got := Get("test.key", false); got != "global-value" {
			t.Errorf("Get() with useLocal=false = %v, want %v", got, "global-value")
		}
	})

	// Test sensitive key handling
	t.Run("Sensitive Keys", func(t *testing.T) {
		// Try to set a sensitive key locally
		err := Set("api.api_key", "secret", false)
		if err == nil {
			t.Error("Expected error when setting sensitive key locally")
		}

		// Set sensitive key globally
		err = Set("api.api_key", "secret", true)
		if err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		// Verify the value is encrypted
		raw := globalData["api.api_key"]
		if raw == "secret" {
			t.Error("Sensitive value should be encrypted")
		}
	})

	// Test config persistence
	t.Run("Config Persistence", func(t *testing.T) {
		// Clear existing data
		globalData = map[string]string{}
		localData = map[string]string{}

		dumpConfig(t, "Initial state for persistence test")

		// Set some values
		if err := Set("persist.test", "persist-value", true); err != nil {
			t.Fatal(err)
		}
		if err := Set("persist.local", "local-value", false); err != nil {
			t.Fatal(err)
		}

		dumpConfig(t, "After setting values for persistence test")

		// Load configs fresh
		globalData = map[string]string{}
		localData = map[string]string{}
		if err := LoadAllConfigs(); err != nil {
			t.Fatalf("LoadAllConfigs() error = %v", err)
		}

		dumpConfig(t, "After reloading configs")

		// Verify values were loaded
		if got := Get("persist.test", false); got != "persist-value" {
			t.Errorf("Get() global = %v, want %v", got, "persist-value")
		}
		if got := Get("persist.local", true); got != "local-value" {
			t.Errorf("Get() local = %v, want %v", got, "local-value")
		}
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
		{"Regular key", "regular.key", false},
		{"Case insensitive", "API.API_KEY", true},
		{"Mixed case", "Github.Token", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Set(tt.key, "test-value", false)
			if tt.sensitive {
				if err == nil {
					t.Error("Expected error for sensitive key")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for non-sensitive key: %v", err)
				}
			}
		})
	}
}
