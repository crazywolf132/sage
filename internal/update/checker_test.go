package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/crazywolf132/sage/internal/gh"
)

// mockGitHubClient implements gh.Client for testing
type mockGitHubClient struct {
	latestVersion string
	err           error
}

func (m *mockGitHubClient) GetLatestRelease() (string, error) {
	return m.latestVersion, m.err
}

// Implement other required methods with no-op implementations
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
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "sage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up test environment
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", tmpDir)
	} else {
		t.Setenv("HOME", tmpDir)
	}

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		ghErr          error
		wantErr        bool
	}{
		{
			name:           "newer version available",
			currentVersion: "1.0.0",
			latestVersion:  "2.0.0",
			wantErr:        false,
		},
		{
			name:           "current version up to date",
			currentVersion: "2.0.0",
			latestVersion:  "2.0.0",
			wantErr:        false,
		},
		{
			name:           "dev version",
			currentVersion: "dev",
			latestVersion:  "2.0.0",
			wantErr:        false,
		},
		{
			name:           "empty version",
			currentVersion: "",
			latestVersion:  "2.0.0",
			wantErr:        false,
		},
		{
			name:           "github error",
			currentVersion: "1.0.0",
			latestVersion:  "",
			ghErr:          errors.New("not found"),
			wantErr:        false, // Should fail silently
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock GitHub client
			mockGH := &mockGitHubClient{
				latestVersion: tt.latestVersion,
				err:           tt.ghErr,
			}

			// Run test
			err := CheckForUpdates(mockGH, tt.currentVersion)

			// Check result
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckForUpdates() error = %v, wantErr %v", err, tt.wantErr)
			}

			// If we expect a newer version, verify the check state was saved
			if tt.latestVersion != "" && tt.currentVersion != "dev" && tt.currentVersion != "" && tt.ghErr == nil {
				configPath, _ := getConfigPath()
				if _, err := os.Stat(configPath); err != nil {
					t.Errorf("Expected check state file to exist at %s", configPath)
				}
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "sage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set up test environment
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", tmpDir)
	} else {
		t.Setenv("HOME", tmpDir)
	}

	// Get config path
	path, err := getConfigPath()
	if err != nil {
		t.Fatalf("getConfigPath() error = %v", err)
	}

	// Verify path
	if runtime.GOOS == "windows" {
		expected := filepath.Join(tmpDir, "sage", "update_check.json")
		if path != expected {
			t.Errorf("getConfigPath() = %v, want %v", path, expected)
		}
	} else {
		expected := filepath.Join(tmpDir, ".config", "sage", "update_check.json")
		if path != expected {
			t.Errorf("getConfigPath() = %v, want %v", path, expected)
		}
	}

	// Verify directory was created
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("Expected directory to exist at %s", dir)
	}
}

func TestShouldCheck(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "sage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "update_check.json")

	tests := []struct {
		name     string
		setupFn  func()
		want     bool
		wantFile bool
	}{
		{
			name: "no file exists",
			setupFn: func() {
				os.Remove(testFile)
			},
			want:     true,
			wantFile: false,
		},
		{
			name: "invalid json",
			setupFn: func() {
				os.WriteFile(testFile, []byte("invalid json"), 0644)
			},
			want:     true,
			wantFile: true,
		},
		{
			name: "check needed (>24h)",
			setupFn: func() {
				state := checkState{
					LastCheck: time.Now().Add(-25 * time.Hour),
					Version:   "1.0.0",
				}
				data, _ := json.Marshal(state)
				os.WriteFile(testFile, data, 0644)
			},
			want:     true,
			wantFile: true,
		},
		{
			name: "check not needed (<24h)",
			setupFn: func() {
				state := checkState{
					LastCheck: time.Now().Add(-23 * time.Hour),
					Version:   "1.0.0",
				}
				data, _ := json.Marshal(state)
				os.WriteFile(testFile, data, 0644)
			},
			want:     false,
			wantFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test case
			tt.setupFn()

			// Run test
			got := shouldCheck(testFile)

			// Check result
			if got != tt.want {
				t.Errorf("shouldCheck() = %v, want %v", got, tt.want)
			}

			// Verify file state
			_, err := os.Stat(testFile)
			exists := err == nil
			if exists != tt.wantFile {
				t.Errorf("File exists = %v, want %v", exists, tt.wantFile)
			}
		})
	}
}

func TestSaveCheckState(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "sage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "update_check.json")
	testVersion := "1.0.0"

	// Save state
	err = saveCheckState(testFile, testVersion)
	if err != nil {
		t.Fatalf("saveCheckState() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("Expected file to exist at %s", testFile)
	}

	// Read and verify contents
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	var state checkState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Failed to unmarshal test file: %v", err)
	}

	if state.Version != testVersion {
		t.Errorf("Saved version = %v, want %v", state.Version, testVersion)
	}

	// Check that LastCheck is recent
	if time.Since(state.LastCheck) > time.Minute {
		t.Errorf("LastCheck time is too old: %v", state.LastCheck)
	}
}
