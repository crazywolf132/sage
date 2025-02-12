package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/crazywolf132/sage/internal/gh"
)

// mockGitHubClient implements gh.Client interface for testing
type mockGitHubClient struct {
	latestVersion string
	err           error
}

func (m *mockGitHubClient) GetLatestRelease() (string, error) {
	return m.latestVersion, m.err
}

// Implement other required interface methods with empty implementations
func (m *mockGitHubClient) CreatePR(title, body, head, base string, draft bool) (*gh.PullRequest, error) {
	return nil, nil
}
func (m *mockGitHubClient) ListPRs(state string) ([]gh.PullRequest, error) {
	return nil, nil
}
func (m *mockGitHubClient) MergePR(num int, method string) error {
	return nil
}
func (m *mockGitHubClient) ClosePR(num int) error {
	return nil
}
func (m *mockGitHubClient) GetPRDetails(num int) (*gh.PullRequest, error) {
	return nil, nil
}
func (m *mockGitHubClient) CheckoutPR(num int) (string, error) {
	return "", nil
}
func (m *mockGitHubClient) ListPRUnresolvedThreads(prNum int) ([]gh.UnresolvedThread, error) {
	return nil, nil
}
func (m *mockGitHubClient) GetPRTemplate() (string, error) {
	return "", nil
}
func (m *mockGitHubClient) AddLabels(prNumber int, labels []string) error {
	return nil
}
func (m *mockGitHubClient) RequestReviewers(prNumber int, reviewers []string) error {
	return nil
}
func (m *mockGitHubClient) GetPRForBranch(branchName string) (*gh.PullRequest, error) {
	return nil, nil
}
func (m *mockGitHubClient) UpdatePR(num int, pr *gh.PullRequest) error {
	return nil
}

func TestCheckForUpdates(t *testing.T) {
	// Create temporary directory for test config
	tmpDir, err := os.MkdirTemp("", "sage-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up test environment
	origAppData := os.Getenv("APPDATA")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("APPDATA", origAppData)
		os.Setenv("HOME", origHome)
	}()
	os.Setenv("APPDATA", tmpDir)
	os.Setenv("HOME", tmpDir)

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		checkState     *checkState
		wantCheck      bool
	}{
		{
			name:           "New version available",
			currentVersion: "1.0.0",
			latestVersion:  "1.1.0",
			wantCheck:      true,
		},
		{
			name:           "Current version up to date",
			currentVersion: "1.0.0",
			latestVersion:  "1.0.0",
			wantCheck:      true,
		},
		{
			name:           "Dev version",
			currentVersion: "dev",
			latestVersion:  "1.0.0",
			wantCheck:      false,
		},
		{
			name:           "Empty version",
			currentVersion: "",
			latestVersion:  "1.0.0",
			wantCheck:      false,
		},
		{
			name:           "Recent check",
			currentVersion: "1.0.0",
			latestVersion:  "1.1.0",
			checkState: &checkState{
				LastCheck: time.Now(),
				Version:   "1.1.0",
			},
			wantCheck: false,
		},
		{
			name:           "Old check",
			currentVersion: "1.0.0",
			latestVersion:  "1.1.0",
			checkState: &checkState{
				LastCheck: time.Now().Add(-25 * time.Hour),
				Version:   "1.0.0",
			},
			wantCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing state
			configPath, err := getConfigPath()
			if err != nil {
				t.Fatal(err)
			}
			os.Remove(configPath)

			// Set up initial state if provided
			if tt.checkState != nil {
				data, err := json.MarshalIndent(tt.checkState, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatal(err)
				}
			}

			// Create mock client
			client := &mockGitHubClient{
				latestVersion: tt.latestVersion,
			}

			// Run update check
			err = CheckForUpdates(client, tt.currentVersion)
			if err != nil {
				t.Errorf("CheckForUpdates() error = %v", err)
			}

			// Verify check state was saved if a check was performed
			if tt.wantCheck {
				data, err := os.ReadFile(configPath)
				if err != nil {
					t.Fatal(err)
				}

				var state checkState
				if err := json.Unmarshal(data, &state); err != nil {
					t.Fatal(err)
				}

				if state.Version != tt.latestVersion {
					t.Errorf("Saved version = %v, want %v", state.Version, tt.latestVersion)
				}
				if time.Since(state.LastCheck) > time.Minute {
					t.Error("LastCheck time not updated")
				}
			}
		})
	}
}

func TestShouldCheck(t *testing.T) {
	// Create temporary directory for test config
	tmpDir, err := os.MkdirTemp("", "sage-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "update_check.json")

	tests := []struct {
		name      string
		lastCheck time.Time
		want      bool
	}{
		{
			name:      "No previous check",
			lastCheck: time.Time{},
			want:      true,
		},
		{
			name:      "Recent check",
			lastCheck: time.Now(),
			want:      false,
		},
		{
			name:      "Old check",
			lastCheck: time.Now().Add(-25 * time.Hour),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing state
			os.Remove(configPath)

			if !tt.lastCheck.IsZero() {
				state := checkState{
					LastCheck: tt.lastCheck,
					Version:   "1.0.0",
				}
				data, err := json.MarshalIndent(state, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(configPath, data, 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := shouldCheck(configPath)
			if got != tt.want {
				t.Errorf("shouldCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Create temporary directory for test config
	tmpDir, err := os.MkdirTemp("", "sage-update-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create necessary subdirectories
	if err := os.MkdirAll(filepath.Join(tmpDir, "AppData", "Roaming"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".config"), 0755); err != nil {
		t.Fatal(err)
	}

	// Save original environment
	origAppData := os.Getenv("APPDATA")
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("APPDATA", origAppData)
		os.Setenv("HOME", origHome)
	}()

	// Platform-specific tests
	var tests []struct {
		name     string
		appData  string
		home     string
		wantPath string
		wantErr  bool
	}

	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			appData  string
			home     string
			wantPath string
			wantErr  bool
		}{
			name:     "Windows path",
			appData:  filepath.Join(tmpDir, "AppData", "Roaming"),
			home:     tmpDir,
			wantPath: filepath.Join(tmpDir, "AppData", "Roaming", "sage", "update_check.json"),
		})
	} else {
		tests = append(tests, struct {
			name     string
			appData  string
			home     string
			wantPath string
			wantErr  bool
		}{
			name:     "Unix path",
			home:     tmpDir,
			wantPath: filepath.Join(tmpDir, ".config", "sage", "update_check.json"),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.appData != "" {
				os.Setenv("APPDATA", tt.appData)
			} else {
				os.Unsetenv("APPDATA") // Ensure APPDATA is not set for Unix tests
			}
			os.Setenv("HOME", tt.home)

			got, err := getConfigPath()
			if (err != nil) != tt.wantErr {
				t.Errorf("getConfigPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantPath {
				t.Errorf("getConfigPath() = %v, want %v", got, tt.wantPath)
			}
		})
	}
}
