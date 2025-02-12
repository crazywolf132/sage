package config

import (
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
}

// KnownGitConfigFeatures maps experimental feature names to their git config requirements
var KnownGitConfigFeatures = map[string]GitConfigFeature{
	"rerere": {
		Key:      "rerere.enabled",
		Value:    "true",
		Default:  "false",
		Global:   false, // We don't want to affect git's global config
		Required: true,
		SageWide: true, // When enabled globally in Sage, apply to all repos where Sage is used
	},
	// Add more git-config-based experimental features here
}

// SyncGitConfigFeatures ensures git config settings match the experimental features state
func SyncGitConfigFeatures() error {
	g := git.NewShellGit()

	for featureName, feature := range KnownGitConfigFeatures {
		if !feature.Required {
			continue
		}

		// Check if feature is enabled either globally or locally
		enabled := IsExperimentalFeatureEnabled(featureName)

		// For SageWide features, also check global Sage config when in a repo
		if !enabled && feature.SageWide {
			// If we're in a repo, check global config
			repo, err := g.IsRepo()
			if err == nil && repo {
				enabled = IsExperimentalFeatureEnabled(featureName)
			}
		}

		value := feature.Default
		if enabled {
			value = feature.Value
		}

		// Set the git config value
		if err := g.SetConfig(feature.Key, value, feature.Global); err != nil {
			return err
		}
	}

	return nil
}
