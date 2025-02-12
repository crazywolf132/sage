package version

import (
	"runtime/debug"
	"strings"
	"sync"
	"testing"
)

// mockBuildInfo is used to simulate different build info scenarios
type mockBuildInfo struct {
	mainVersion string
	settings    []debug.BuildSetting
}

func (m mockBuildInfo) toBuildInfo() *debug.BuildInfo {
	return &debug.BuildInfo{
		Main: debug.Module{
			Path:    "github.com/crazywolf132/sage",
			Version: m.mainVersion,
		},
		Settings: m.settings,
	}
}

func TestGet(t *testing.T) {
	// Reset package state before each test
	defer func() {
		Version = ""
		versionString = ""
		once = sync.Once{}
		buildInfoFunc = debug.ReadBuildInfo
	}()

	t.Run("Release version", func(t *testing.T) {
		Version = "1.2.3"
		got := Get()
		if got != "1.2.3" {
			t.Errorf("Get() = %v, want %v", got, "1.2.3")
		}
	})

	t.Run("Cached value", func(t *testing.T) {
		Version = "1.2.3"
		first := Get()
		Version = "2.0.0" // Change should not affect cached value
		second := Get()
		if first != second {
			t.Errorf("Get() returned different values: first=%v, second=%v", first, second)
		}
	})
}

func TestDetermineVersion(t *testing.T) {
	// Save original buildInfoFunc and restore after tests
	originalBuildInfoFunc := buildInfoFunc
	defer func() {
		buildInfoFunc = originalBuildInfoFunc
		Version = ""
	}()

	tests := []struct {
		name    string
		version string
		mock    mockBuildInfo
		want    string
	}{
		{
			name:    "Release version",
			version: "1.2.3",
			want:    "1.2.3",
		},
		{
			name:    "Empty version with main module version",
			version: "",
			mock: mockBuildInfo{
				mainVersion: "v2.3.4",
			},
			want: "2.3.4",
		},
		{
			name:    "Development build with VCS info",
			version: "",
			mock: mockBuildInfo{
				mainVersion: "(devel)",
				settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abcdef1234567890"},
					{Key: "vcs.time", Value: "2024-01-01T12:00:00Z"},
				},
			},
			want: "dev-abcdef1-2024-01-01T12:00:00Z",
		},
		{
			name:    "Development build without VCS info",
			version: "",
			mock: mockBuildInfo{
				mainVersion: "(devel)",
			},
			want: "0.0.0-dev",
		},
		{
			name:    "No version info",
			version: "",
			mock:    mockBuildInfo{},
			want:    "0.0.0-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			Version = tt.version
			buildInfoFunc = func() (*debug.BuildInfo, bool) {
				if tt.version != "" {
					return nil, false
				}
				info := tt.mock.toBuildInfo()
				return info, true
			}

			got := determineVersion()
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("determineVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Variables to allow mocking in tests
var readBuildInfo = debug.ReadBuildInfo
