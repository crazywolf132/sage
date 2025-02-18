package version

import (
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

// Mock build info for testing
type mockBuildInfo struct {
	mainModule    debug.Module
	buildSettings []debug.BuildSetting
}

func (m *mockBuildInfo) Main() debug.Module {
	return m.mainModule
}

func (m *mockBuildInfo) Settings() []debug.BuildSetting {
	return m.buildSettings
}

// mockBuildInfoForTest is used to provide test data
var mockBuildInfoForTest *mockBuildInfo

// mockReadBuildInfo is a package-level variable for mocking debug.ReadBuildInfo
var mockReadBuildInfo = func() (*debug.BuildInfo, bool) {
	if mockBuildInfoForTest == nil {
		return nil, false
	}
	return &debug.BuildInfo{
		Main:     mockBuildInfoForTest.Main(),
		Settings: mockBuildInfoForTest.Settings(),
	}, true
}

func init() {
	// Override the package's readBuildInfo to use our mock
	readBuildInfo = mockReadBuildInfo
}

// resetOnce resets the sync.Once to its initial state
func resetOnce(o *sync.Once) {
	// Use reflection to reset the sync.Once to its initial state
	v := reflect.ValueOf(o).Elem()
	v.FieldByName("done").SetUint(0)
}

func TestGet(t *testing.T) {
	// Save original values
	origOnce := once
	origVersionString := versionString
	origVersion := Version
	defer func() {
		// Restore original values after test
		once = origOnce
		versionString = origVersionString
		Version = origVersion
	}()

	// Reset package state before test
	once = sync.Once{}
	versionString = "" // Reset cached version
	Version = ""       // Reset global Version

	// Set up mock build info for a development version
	mockBuildInfoForTest = &mockBuildInfo{
		mainModule: debug.Module{
			Path:    "github.com/crazywolf132/sage",
			Version: "(devel)",
		},
		buildSettings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef1234567"},
			{Key: "vcs.time", Value: time.Now().Format(time.RFC3339)},
		},
	}

	// First call should determine version
	v1 := Get()
	if v1 == "" {
		t.Error("Get() returned empty string")
	}
	if !strings.HasPrefix(v1, "dev-") {
		t.Errorf("Get() = %v, want prefix 'dev-'", v1)
	}

	// Second call should return cached version
	v2 := Get()
	if v1 != v2 {
		t.Errorf("Get() returned different values: %s != %s", v1, v2)
	}
}

func TestDetermineVersion(t *testing.T) {
	// Save original values
	origOnce := once
	origVersionString := versionString
	origVersion := Version
	defer func() {
		// Restore original values after test
		once = origOnce
		versionString = origVersionString
		Version = origVersion
	}()

	tests := []struct {
		name           string
		setupFn        func()
		expectedPrefix string
	}{
		{
			name: "with ldflags version",
			setupFn: func() {
				Version = "1.2.3"
				mockBuildInfoForTest = nil
			},
			expectedPrefix: "1.2.3",
		},
		{
			name: "development version",
			setupFn: func() {
				Version = ""
				mockBuildInfoForTest = nil
			},
			expectedPrefix: "0.0.0-dev",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset package state
			once = sync.Once{}
			versionString = ""

			// Setup test case
			tt.setupFn()

			// Run test
			result := determineVersion()

			// Check result
			if result != tt.expectedPrefix {
				t.Errorf("determineVersion() = %v, want %v", result, tt.expectedPrefix)
			}
		})
	}
}

func TestDetermineVersionWithBuildInfo(t *testing.T) {
	// Save original values
	origOnce := once
	origVersionString := versionString
	origVersion := Version
	defer func() {
		// Restore original values after test
		once = origOnce
		versionString = origVersionString
		Version = origVersion
	}()

	tests := []struct {
		name           string
		buildInfo      *mockBuildInfo
		expectedPrefix string
	}{
		{
			name: "with main module version",
			buildInfo: &mockBuildInfo{
				mainModule: debug.Module{
					Path:    "github.com/crazywolf132/sage",
					Version: "v1.2.3",
				},
			},
			expectedPrefix: "1.2.3",
		},
		{
			name: "with vcs information",
			buildInfo: &mockBuildInfo{
				mainModule: debug.Module{
					Path:    "github.com/crazywolf132/sage",
					Version: "(devel)",
				},
				buildSettings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "abcdef1234567"},
					{Key: "vcs.time", Value: time.Now().Format(time.RFC3339)},
				},
			},
			expectedPrefix: "dev-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset package state
			once = sync.Once{}
			versionString = ""
			Version = ""
			mockBuildInfoForTest = tt.buildInfo

			// Run test
			result := determineVersion()

			// Verify result
			if tt.expectedPrefix != "" && !strings.HasPrefix(result, tt.expectedPrefix) {
				t.Errorf("determineVersion() = %v, want prefix %v", result, tt.expectedPrefix)
			}
		})
	}
}

func TestDetermineVersionFallback(t *testing.T) {
	// Save original values
	origOnce := once
	origVersionString := versionString
	origVersion := Version
	defer func() {
		// Restore original values after test
		once = origOnce
		versionString = origVersionString
		Version = origVersion
	}()

	// Reset package state
	once = sync.Once{}
	versionString = ""
	Version = ""
	mockBuildInfoForTest = nil

	// Run test
	result := determineVersion()

	// Verify fallback version
	expected := "0.0.0-dev"
	if result != expected {
		t.Errorf("determineVersion() = %v, want %v", result, expected)
	}
}
