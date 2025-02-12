package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/crazywolf132/sage/internal/git"
)

// GitConfigFeature represents an experimental feature that requires git config changes
type GitConfigFeature struct {
	Key      string // git config key
	Value    string // value when enabled
	Default  string // value when disabled
	Global   bool   // whether this should be set in git's global config
	Required bool   // whether this feature requires the git config to be set
	// SageWide indicates if the feature, when enabled globally in Sage,
	// should apply to all repositories where Sage is used
	SageWide bool
	// IsCommand indicates if this is a command that needs to be run rather than a config setting
	IsCommand bool
	// Command is the git command to run when enabling the feature
	Command string
	// DisableCommand is the git command to run when disabling the feature
	DisableCommand string
	// StateFile is the name of the file to store state in (relative to global config dir)
	StateFile string
}

// KnownGitConfigFeatures maps experimental feature names to their git config requirements
var KnownGitConfigFeatures = map[string]GitConfigFeature{
	"rerere": {
		Key:      "rerere.enabled",
		Value:    "true",
		Default:  "false",
		Global:   false,
		Required: true,
		SageWide: true,
	},
	"commit-graph": {
		Key:      "fetch.writeCommitGraph",
		Value:    "true",
		Default:  "false",
		Global:   false,
		Required: true,
		SageWide: true,
	},
	"fsmonitor": {
		Key:      "core.fsmonitor",
		Value:    "true",
		Default:  "false",
		Global:   false,
		Required: true,
		SageWide: true,
	},
	"maintenance": {
		IsCommand:      true,
		Command:        "maintenance start",
		DisableCommand: "maintenance stop",
		StateFile:      "maintenance_repos.json",
		Required:       true,
		SageWide:       true, // Allow global enablement
	},
}

// maintenanceState tracks which repositories have maintenance enabled
type maintenanceState struct {
	EnabledRepos map[string]bool `json:"enabled_repos"` // map of repo paths to enabled state
}

// loadMaintenanceState loads the state of maintenance-enabled repositories
func loadMaintenanceState() (maintenanceState, error) {
	state := maintenanceState{
		EnabledRepos: make(map[string]bool),
	}

	configPath, err := globalPath()
	if err != nil {
		return state, err
	}

	stateFile := filepath.Join(filepath.Dir(configPath), "maintenance_repos.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}

	err = json.Unmarshal(data, &state)
	return state, err
}

// saveMaintenanceState saves the state of maintenance-enabled repositories
func saveMaintenanceState(state maintenanceState) error {
	configPath, err := globalPath()
	if err != nil {
		return err
	}

	stateFile := filepath.Join(filepath.Dir(configPath), "maintenance_repos.json")
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

// isMaintenanceEnabled checks if maintenance is enabled for the current repository
func isMaintenanceEnabled(repoPath string) bool {
	state, err := loadMaintenanceState()
	if err != nil {
		return false
	}
	return state.EnabledRepos[repoPath]
}

// setMaintenanceEnabled sets the maintenance state for the current repository
func setMaintenanceEnabled(repoPath string, enabled bool) error {
	state, err := loadMaintenanceState()
	if err != nil {
		return err
	}

	if enabled {
		state.EnabledRepos[repoPath] = true
	} else {
		delete(state.EnabledRepos, repoPath)
	}

	return saveMaintenanceState(state)
}

// GetExperimentalFeatures returns all known experimental features
func GetExperimentalFeatures() map[string]GitConfigFeature {
	return KnownGitConfigFeatures
}

// SyncGitConfigFeatures ensures git config settings match the experimental features state
func SyncGitConfigFeatures() error {
	g := git.NewShellGit()

	for featureName, feature := range KnownGitConfigFeatures {
		if !feature.Required {
			continue
		}

		if feature.IsCommand {
			if featureName == "maintenance" {
				// Handle maintenance feature specially
				repoPath, err := g.GetRepoPath()
				if err != nil {
					continue // Not in a repo
				}

				// Check if maintenance should be enabled (either globally or locally)
				globalEnabled := Get("experimental."+featureName, false) == "true"
				localEnabled := Get("experimental."+featureName, true) == "true"
				shouldBeEnabled := globalEnabled || localEnabled
				currentlyEnabled := isMaintenanceEnabled(repoPath)

				if shouldBeEnabled && !currentlyEnabled {
					// Enable maintenance for this repo
					if _, err := g.Run(strings.Split(feature.Command, " ")...); err != nil {
						return err
					}
					if err := setMaintenanceEnabled(repoPath, true); err != nil {
						return err
					}
				} else if !shouldBeEnabled && currentlyEnabled {
					// Disable maintenance for this repo
					if _, err := g.Run(strings.Split(feature.DisableCommand, " ")...); err != nil {
						return err
					}
					if err := setMaintenanceEnabled(repoPath, false); err != nil {
						return err
					}
				}
			}
			continue
		}

		// Handle regular config-based features
		enabled := IsExperimentalFeatureEnabled(featureName)

		if !enabled && feature.SageWide {
			repo, err := g.IsRepo()
			if err == nil && repo {
				enabled = IsExperimentalFeatureEnabled(featureName)
			}
		}

		value := feature.Default
		if enabled {
			value = feature.Value
		}

		if err := g.SetConfig(feature.Key, value, feature.Global); err != nil {
			return err
		}
	}

	return nil
}
