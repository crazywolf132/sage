package undo

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Operation represents a Git operation that can be undone
type Operation struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Command     string    `json:"command"`
	Timestamp   time.Time `json:"timestamp"`
	Ref         string    `json:"ref"`      // Git reference (commit hash, branch name, etc.)
	Category    string    `json:"category"` // e.g., "commit", "merge", "rebase", etc.
	Metadata    struct {
		Files    []string          `json:"files,omitempty"`     // Affected files
		Branch   string            `json:"branch,omitempty"`    // Current branch
		Message  string            `json:"message,omitempty"`   // Commit message if applicable
		Extra    map[string]string `json:"extra,omitempty"`     // Additional metadata
		Stashed  bool              `json:"stashed,omitempty"`   // Whether changes were stashed
		StashRef string            `json:"stash_ref,omitempty"` // Reference to stash if applicable
	} `json:"metadata"`
}

// History manages the undo history for the repository
type History struct {
	Operations []Operation `json:"operations"`
	MaxSize    int         `json:"max_size"` // Maximum number of operations to track
}

// NewHistory creates a new history tracker
func NewHistory() *History {
	return &History{
		Operations: make([]Operation, 0),
		MaxSize:    100, // Default to tracking last 100 operations
	}
}

// AddOperation adds a new operation to the history
func (h *History) AddOperation(op Operation) {
	// Add new operation at the beginning
	h.Operations = append([]Operation{op}, h.Operations...)

	// Trim if exceeding max size
	if len(h.Operations) > h.MaxSize {
		h.Operations = h.Operations[:h.MaxSize]
	}
}

// GetOperations returns operations filtered by category and/or time range
func (h *History) GetOperations(category string, since time.Time) []Operation {
	var filtered []Operation
	for _, op := range h.Operations {
		if (category == "" || op.Category == category) &&
			(since.IsZero() || op.Timestamp.After(since)) {
			filtered = append(filtered, op)
		}
	}
	return filtered
}

// Save persists the history to disk
func (h *History) Save(repoPath string) error {
	historyPath := filepath.Join(repoPath, ".sage", "undo_history.json")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(historyPath), 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}

// Load reads the history from disk
func (h *History) Load(repoPath string) error {
	historyPath := filepath.Join(repoPath, ".sage", "undo_history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No history file yet, start fresh
			return nil
		}
		return fmt.Errorf("failed to read history file: %w", err)
	}

	if err := json.Unmarshal(data, h); err != nil {
		return fmt.Errorf("failed to parse history file: %w", err)
	}

	return nil
}

// Clear removes all operations from history
func (h *History) Clear() {
	h.Operations = make([]Operation, 0)
}
