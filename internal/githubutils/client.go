package githubutils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// DefaultClient is the default HTTP client used for GitHub API requests
	DefaultClient = http.DefaultClient
	// BaseURL is the base URL for GitHub API requests (can be overridden for testing)
	BaseURL = "https://api.github.com"
)

// CreatePullRequest calls GitHub's API to open a new PR.
func CreatePullRequest(token, owner, repo string, body CreatePRParams) (*PullRequest, error) {
	if token == "" {
		return nil, errors.New("no GitHub token provided")
	}

	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls", BaseURL, owner, repo)

	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CreatePRParams: %w", err)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var result PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub response: %w", err)
	}

	return &result, nil
}

// ListPullRequests fetches open PRs (optionally you can add state=all/closed, etc.).
func ListPullRequests(token, owner, repo string, state string) ([]PullRequest, error) {
	if token == "" {
		return nil, errors.New("no GitHub token provided")
	}

	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls?state=%s", BaseURL, owner, repo, state)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var prs []PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}
	return prs, nil
}

// GetPullRequest retrieves a specific PR by number.
func GetPullRequest(token, owner, repo string, number int) (*PullRequest, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", BaseURL, owner, repo, number)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var pr PullRequest
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// ClosePullRequest updates a PR's state to 'closed'.
func ClosePullRequest(token, owner, repo string, number int) error {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", BaseURL, owner, repo, number)

	updateBody := map[string]string{
		"state": "closed",
	}
	payload, _ := json.Marshal(updateBody)

	req, err := http.NewRequest("PATCH", apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	return nil
}

// MergePullRequest merges a PR. You can support "merge", "squash", or "rebase" strategies.
func MergePullRequest(token, owner, repo string, number int, method string) error {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", BaseURL, owner, repo, number)

	body := map[string]string{
		"merge_method": method, // "merge", "squash", or "rebase"
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequest("PUT", apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 405 {
		return errors.New("merge is not allowed by GitHub (possibly checks not passing)")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	return nil
}

// CheckoutPullRequest checks out a PR branch locally using the PR's head reference.
func CheckoutPullRequest(token, owner, repo string, number int) error {
	// First, get the PR details to get the head reference
	pr, err := GetPullRequest(token, owner, repo, number)
	if err != nil {
		return fmt.Errorf("failed to get PR details: %w", err)
	}

	// Fetch the PR's head reference
	fetchCmd := exec.Command("git", "fetch", "origin", pr.Head.Ref)
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch PR %d: %w", number, err)
	}

	// Checkout the fetched reference
	checkoutCmd := exec.Command("git", "checkout", pr.Head.Ref)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", pr.Head.Ref, err)
	}
	return nil
}

// FindRepoOwnerAndName uses "git remote get-url origin" to parse "owner/repo" from GitHub URL.
func FindRepoOwnerAndName() (string, string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", err
	}
	originURL := strings.TrimSpace(string(out))

	// strip .git suffix
	originURL = strings.TrimSuffix(originURL, ".git")

	// Regex to handle both SSH and HTTPS
	re := regexp.MustCompile(`(?i)(?:github\.com[:/])([^/]+)/([^/]+)$`)
	matches := re.FindStringSubmatch(originURL)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("invalid GitHub remote URL: %s", originURL)
	}
	return matches[1], matches[2], nil
}

// GetPullRequestTemplate fetches the PR template from the repository.
// It checks for templates in the following locations (in order):
// 1. .github/PULL_REQUEST_TEMPLATE.md
// 2. .github/pull_request_template.md
// 3. docs/PULL_REQUEST_TEMPLATE.md
// 4. PULL_REQUEST_TEMPLATE.md
func GetPullRequestTemplate(token, owner, repo string) (string, error) {
	templatePaths := []string{
		".github/PULL_REQUEST_TEMPLATE.md",
		".github/pull_request_template.md",
		"docs/PULL_REQUEST_TEMPLATE.md",
		"PULL_REQUEST_TEMPLATE.md",
	}

	for _, path := range templatePaths {
		apiURL := fmt.Sprintf("%s/repos/%s/%s/contents/%s", BaseURL, owner, repo, path)
		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Authorization", "token "+token)

		resp, err := DefaultClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			continue
		}

		var content struct {
			Content  string `json:"content"`
			Encoding string `json:"encoding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
			continue
		}

		// GitHub returns content as base64 encoded
		if content.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(content.Content)
			if err != nil {
				continue
			}
			return string(decoded), nil
		}
	}

	return "", nil // No template found
}
