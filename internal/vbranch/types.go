package vbranch

import (
	"time"
)

// VirtualBranch represents a virtual branch in memory
type VirtualBranch struct {
	Name        string    `json:"name"`
	BaseBranch  string    `json:"base_branch"`
	Created     time.Time `json:"created"`
	LastUpdated time.Time `json:"last_updated"`
	Changes     []Change  `json:"changes"`
	Active      bool      `json:"active"`
	StashedDiff string    `json:"stashed_diff,omitempty"` // Stashed changes when switching branches
}

// Change represents a single file change in a virtual branch
type Change struct {
	Path      string    `json:"path"`
	Diff      string    `json:"diff"`
	Timestamp time.Time `json:"timestamp"`
	Staged    bool      `json:"staged"`
}

// Manager handles virtual branch operations
type Manager interface {
	// Create a new virtual branch
	CreateVirtualBranch(name string, baseBranch string) (*VirtualBranch, error)

	// List all virtual branches
	ListVirtualBranches() ([]*VirtualBranch, error)

	// Get a specific virtual branch
	GetVirtualBranch(name string) (*VirtualBranch, error)

	// Apply changes from a virtual branch to working directory
	ApplyVirtualBranch(name string) error

	// Unapply changes from a virtual branch
	UnapplyVirtualBranch(name string) error

	// Add a change to a virtual branch
	AddChange(branchName string, change Change) error

	// Remove a change from a virtual branch
	RemoveChange(branchName string, path string) error

	// Convert virtual branch to real Git branch
	MaterializeBranch(name string) error

	// Move changes between virtual branches
	MoveChanges(fromBranch string, toBranch string, paths []string) error

	// Get the currently active virtual branch
	GetActiveBranch() (*VirtualBranch, error)

	// Check if a virtual branch has stashed changes
	HasStashedChanges(name string) (bool, error)

	// Pop stashed changes from a virtual branch
	PopStashedChanges(name string) error

	// Drop stashed changes from a virtual branch
	DropStashedChanges(name string) error
}
