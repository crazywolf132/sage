package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/crazywolf132/sage/internal/ui"
)

const (
	repoAPI = "https://api.github.com/repos/crazywolf132/sage/commits/main"
)

type CommitInfo struct {
	SHA string `json:"sha"`
}

// getUpdateCheckPath returns the path to store the update check file
// On Unix systems, it's stored in /var/tmp
// On Windows, it's stored in %LOCALAPPDATA%
func getUpdateCheckPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("LOCALAPPDATA")
		if appData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable not set")
		}
		return filepath.Join(appData, "sage_update_check"), nil
	}

	// For Unix-like systems, use /var/tmp for system-wide access
	return "/var/tmp/sage_update_check", nil
}

// CheckForUpdates checks if there are any updates available for sage
// It only checks once every 24 hours to avoid unnecessary API calls
func CheckForUpdates() error {
	checkFile, err := getUpdateCheckPath()
	if err != nil {
		return fmt.Errorf("failed to get update check path: %w", err)
	}

	shouldCheck, lastKnownSHA := shouldCheckForUpdates(checkFile)
	if !shouldCheck {
		return nil
	}

	// Get the latest commit SHA from GitHub
	latestSHA, err := getLatestCommitSHA()
	if err != nil {
		return fmt.Errorf("failed to get latest commit: %w", err)
	}

	// Update the check file with current time and SHA
	if err := writeCheckFile(checkFile, latestSHA); err != nil {
		return fmt.Errorf("failed to update check file: %w", err)
	}

	// If we have a different SHA and it's not the first run (empty lastKnownSHA)
	if lastKnownSHA != "" && latestSHA != lastKnownSHA {
		fmt.Printf("\n%s %s\n",
			ui.ColoredText("●", ui.Yellow),
			ui.ColoredText("A new version of sage is available!", ui.White))
		fmt.Printf("   Run %s to update\n\n",
			ui.ColoredText("go install github.com/crazywolf132/sage@latest", ui.Blue))
	}

	return nil
}

func shouldCheckForUpdates(checkFile string) (bool, string) {
	data, err := os.ReadFile(checkFile)
	if err != nil {
		return true, "" // First run or error reading file
	}

	var info struct {
		LastCheck time.Time `json:"last_check"`
		SHA       string    `json:"sha"`
	}

	if err := json.Unmarshal(data, &info); err != nil {
		return true, "" // Invalid file format
	}

	// Check if 24 hours have passed
	if time.Since(info.LastCheck) < 24*time.Hour {
		return false, info.SHA
	}

	return true, info.SHA
}

func getLatestCommitSHA() (string, error) {
	resp, err := http.Get(repoAPI)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var commit CommitInfo
	if err := json.Unmarshal(body, &commit); err != nil {
		return "", err
	}

	return commit.SHA, nil
}

func writeCheckFile(checkFile, sha string) error {
	info := struct {
		LastCheck time.Time `json:"last_check"`
		SHA       string    `json:"sha"`
	}{
		LastCheck: time.Now(),
		SHA:       sha,
	}

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	return os.WriteFile(checkFile, data, 0644)
}
