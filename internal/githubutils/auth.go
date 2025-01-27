package githubutils

import (
	"os"
	"os/exec"
	"strings"
)

// GetGitHubToken tries "gh auth token" if GH CLI is installed; otherwise checks env vars.
func GetGitHubToken() (string, error) {
	ghPath, err := exec.LookPath("gh")
	if err == nil && ghPath != "" {
		out, err := exec.Command("gh", "auth", "token").Output()
		if err == nil {
			token := strings.TrimSpace(string(out))
			if token != "" {
				return token, nil
			}
		}
	}

	// Fallback to environment variable
	token := os.Getenv("SAGE_GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}

	return token, nil
}
