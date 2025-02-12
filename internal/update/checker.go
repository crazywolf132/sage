package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/gh"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/hashicorp/go-version"
)

type checkState struct {
	LastCheck time.Time `json:"last_check"`
	Version   string    `json:"version"`
}

// CheckForUpdates checks if a newer version of sage is available
func CheckForUpdates(ghc gh.Client, currentVersion string) error {
	// Skip check for dev versions
	if currentVersion == "dev" || currentVersion == "" {
		return nil
	}

	// Get config file path
	configPath, err := getConfigPath()
	if err != nil {
		return nil // Silently fail if we can't get config path
	}

	// Check if we need to perform an update check
	if !shouldCheck(configPath) {
		return nil
	}

	// Get current version
	current, err := version.NewVersion(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return nil // Silently fail for dev versions
	}

	// Get latest version from GitHub
	latestVersion, err := ghc.GetLatestRelease()
	if err != nil {
		return nil // Silently fail if we can't reach GitHub
	}

	// Parse latest version
	latest, err := version.NewVersion(latestVersion)
	if err != nil {
		return nil // Silently fail for invalid versions
	}

	// Save check state
	_ = saveCheckState(configPath, latestVersion)

	// Compare versions
	if latest.GreaterThan(current) {
		ui.Info(fmt.Sprintf("A new version of sage is available: %s → %s", current, latest))
		ui.Info("To update, run: go install github.com/crazywolf132/sage@latest")
		fmt.Println() // Add a blank line for better readability
	}

	return nil
}

func getConfigPath() (string, error) {
	var configDir string
	if runtime.GOOS == "windows" {
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(configDir, "sage")
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(homeDir, ".config", "sage")
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "update_check.json"), nil
}

func shouldCheck(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return true // Check if we can't read the file
	}

	var state checkState
	if err := json.Unmarshal(data, &state); err != nil {
		return true // Check if we can't parse the file
	}

	// Check if 24 hours have passed since last check
	return time.Since(state.LastCheck) >= 24*time.Hour
}

func saveCheckState(path string, version string) error {
	state := checkState{
		LastCheck: time.Now(),
		Version:   version,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
