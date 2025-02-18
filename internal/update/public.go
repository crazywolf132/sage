package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/crazywolf132/sage/internal/ui"
	"github.com/hashicorp/go-version"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdatesPublic checks if a newer version of sage is available using the public GitHub API
func CheckForUpdatesPublic(currentVersion string) error {
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

	// Get latest version from GitHub's public API
	latestVersion, err := getLatestReleasePublic()
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

// getLatestReleasePublic gets the latest release version using GitHub's public API
func getLatestReleasePublic() (string, error) {
	url := "https://api.github.com/repos/crazywolf132/sage/releases/latest"

	// Create a client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Add required headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "sage-cli")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the response
	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}
