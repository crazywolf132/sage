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
	"sort"
	"strings"
	"time"
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

	// Create and checkout a new branch with the PR number
	branchName := fmt.Sprintf("pr-%d", number)
	checkoutCmd := exec.Command("git", "checkout", "-b", branchName, "origin/"+pr.Head.Ref)
	if err := checkoutCmd.Run(); err != nil {
		// If branch exists, try to check it out directly
		checkoutCmd = exec.Command("git", "checkout", branchName)
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
		}
	}

	// Set up tracking branch
	trackCmd := exec.Command("git", "branch", "--set-upstream-to", "origin/"+pr.Head.Ref, branchName)
	if err := trackCmd.Run(); err != nil {
		return fmt.Errorf("failed to set upstream branch: %w", err)
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

// GetPullRequestDetails fetches comprehensive information about a PR including reviews, checks, and timeline
func GetPullRequestDetails(token, owner, repo string, number int) (*PullRequestDetails, error) {
	// First get the basic PR info
	pr, err := GetPullRequest(token, owner, repo, number)
	if err != nil {
		return nil, err
	}

	details := &PullRequestDetails{
		Number:  pr.Number,
		HTMLURL: pr.HTMLURL,
		Title:   pr.Title,
		State:   pr.State,
		Body:    pr.Body,
		Draft:   pr.Draft,
		Head:    pr.Head,
		Base:    pr.Base,
		Merged:  pr.Merged,
	}

	// Get reviews
	reviews, err := getPRReviews(token, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR reviews: %w", err)
	}
	details.Reviews = reviews

	// Get check runs
	checks, err := getPRChecks(token, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR checks: %w", err)
	}
	details.Checks = checks

	// Get timeline events
	events, err := getPRTimeline(token, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR timeline: %w", err)
	}
	details.Timeline = events

	return details, nil
}

// GetCurrentBranchPR finds a PR associated with the current branch
func GetCurrentBranchPR(token, owner, repo string) (*PullRequest, error) {
	currentBranch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}

	// List open PRs
	prs, err := ListPullRequests(token, owner, repo, "open")
	if err != nil {
		return nil, err
	}

	branchName := strings.TrimSpace(string(currentBranch))
	for _, pr := range prs {
		if pr.Head.Ref == branchName {
			return &pr, nil
		}
	}

	return nil, nil
}

func getPRReviews(token, owner, repo string, number int) ([]PRReview, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/reviews", BaseURL, owner, repo, number)
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

	var reviews []PRReview
	if err := json.NewDecoder(resp.Body).Decode(&reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

func getPRChecks(token, owner, repo string, number int) ([]PRCheck, error) {
	apiURL := fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs", BaseURL, owner, repo, "HEAD")
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var response struct {
		CheckRuns []PRCheck `json:"check_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.CheckRuns, nil
}

func getPRTimeline(token, owner, repo string, number int) ([]PREvent, error) {
	// Get commits first
	commitsURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/commits", BaseURL, owner, repo, number)
	req, err := http.NewRequest("GET", commitsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	type commitInfo struct {
		SHA    string `json:"sha"`
		Commit struct {
			Message string `json:"message"`
			Author  struct {
				Date string `json:"date"`
			} `json:"author"`
		} `json:"commit"`
		Author struct {
			Login string `json:"login"`
		} `json:"author"`
	}

	var commits []commitInfo
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, err
	}

	// Convert commits to events
	var events []PREvent
	for i := len(commits) - 1; i >= 0 && i >= len(commits)-5; i-- {
		commit := commits[i]
		createdAt, _ := time.Parse(time.RFC3339, commit.Commit.Author.Date)
		events = append(events, PREvent{
			Event: "committed",
			Actor: struct {
				Login string `json:"login"`
			}{Login: commit.Author.Login},
			CreatedAt: createdAt,
			Message:   commit.Commit.Message,
			SHA:       commit.SHA[:7], // Short SHA
		})
	}

	return events, nil
}

// GetPRReviewComments fetches all review comments for a PR and organizes them into threads
func GetPRReviewComments(token, owner, repo string, number int) ([]PRReviewComment, error) {
	// Get review comments (these are the inline comments)
	apiURL := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/comments", BaseURL, owner, repo, number)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.comfort-fade-preview+json") // For thread information

	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var comments []PRReviewComment
	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return nil, err
	}

	// Get issue comments (these are the general PR comments)
	issueURL := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", BaseURL, owner, repo, number)
	req, err = http.NewRequest("GET", issueURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)

	resp, err = DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var issueComments []PRReviewComment
		if err := json.NewDecoder(resp.Body).Decode(&issueComments); err == nil {
			comments = append(comments, issueComments...)
		}
	}

	// Sort comments by thread and creation time
	sort.Slice(comments, func(i, j int) bool {
		if comments[i].ThreadID != comments[j].ThreadID {
			return comments[i].ThreadID < comments[j].ThreadID
		}
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})

	return comments, nil
}
