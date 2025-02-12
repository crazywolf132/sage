package config

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
	"repo-ranger": {
		Required: true,
		SageWide: true, // Can be enabled globally for all repos
	},
}
