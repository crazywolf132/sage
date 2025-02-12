package undo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryMigration(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "sage-history-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directories
	oldDir := filepath.Join(tmpDir, ".sage")
	newDir := filepath.Join(tmpDir, ".git", "sage")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test history data
	testHistory := &History{
		Operations: []Operation{
			{
				ID:          "test-id",
				Type:        "commit",
				Description: "test commit",
				Command:     "git commit",
				Timestamp:   time.Now(),
				Ref:         "HEAD",
				Category:    "commit",
			},
		},
		MaxSize: 100,
	}

	// Write test data to old location
	oldPath := filepath.Join(oldDir, "undo_history.json")
	data, err := json.MarshalIndent(testHistory, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Test migration
	h := NewHistory()
	if err := h.Load(tmpDir); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify old file is gone
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old history file still exists")
	}

	// Verify new file exists
	newPath := filepath.Join(newDir, "undo_history.json")
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("New history file not created")
	}

	// Verify data was migrated correctly
	if len(h.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(h.Operations))
	}
	if h.Operations[0].ID != "test-id" {
		t.Errorf("Operation ID = %v, want %v", h.Operations[0].ID, "test-id")
	}
}

func TestHistoryOperations(t *testing.T) {
	h := NewHistory()

	// Test adding operations
	op1 := Operation{
		ID:          "1",
		Type:        "commit",
		Description: "first commit",
		Category:    "commit",
		Timestamp:   time.Now().Add(-1 * time.Hour),
	}
	op2 := Operation{
		ID:          "2",
		Type:        "merge",
		Description: "merge branch",
		Category:    "merge",
		Timestamp:   time.Now(),
	}

	h.AddOperation(op1)
	h.AddOperation(op2)

	// Test size limit
	if len(h.Operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(h.Operations))
	}

	// Test order (newest first)
	if h.Operations[0].ID != "2" {
		t.Errorf("First operation ID = %v, want %v", h.Operations[0].ID, "2")
	}

	// Test filtering by category
	commits := h.GetOperations("commit", time.Time{})
	if len(commits) != 1 {
		t.Errorf("Expected 1 commit operation, got %d", len(commits))
	}
	if commits[0].ID != "1" {
		t.Errorf("Commit operation ID = %v, want %v", commits[0].ID, "1")
	}

	// Test filtering by time
	recent := h.GetOperations("", time.Now().Add(-30*time.Minute))
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent operation, got %d", len(recent))
	}
	if recent[0].ID != "2" {
		t.Errorf("Recent operation ID = %v, want %v", recent[0].ID, "2")
	}

	// Test clear
	h.Clear()
	if len(h.Operations) != 0 {
		t.Errorf("Expected 0 operations after clear, got %d", len(h.Operations))
	}
}

func TestHistorySaveLoad(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "sage-history-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create .git directory
	if err := os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create and save history
	h1 := NewHistory()
	h1.AddOperation(Operation{
		ID:          "test-id",
		Type:        "commit",
		Description: "test commit",
		Category:    "commit",
	})

	if err := h1.Save(tmpDir); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load history in new instance
	h2 := NewHistory()
	if err := h2.Load(tmpDir); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify data
	if len(h2.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(h2.Operations))
	}
	if h2.Operations[0].ID != "test-id" {
		t.Errorf("Operation ID = %v, want %v", h2.Operations[0].ID, "test-id")
	}
}
