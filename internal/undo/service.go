package undo

import (
	"fmt"
	"time"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/google/uuid"
)

// Service handles undo operations
type Service struct {
	git     git.Service
	history *History
}

// NewService creates a new undo service
func NewService(g git.Service) *Service {
	return &Service{
		git:     g,
		history: NewHistory(),
	}
}

// RecordOperation records an operation in the history
func (s *Service) RecordOperation(opType, description, command, category string, metadata Operation) error {
	ref, err := s.git.GetCommitHash("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	op := Operation{
		ID:          uuid.New().String(),
		Type:        opType,
		Description: description,
		Command:     command,
		Timestamp:   time.Now(),
		Ref:         ref,
		Category:    category,
		Metadata:    metadata.Metadata,
	}

	s.history.AddOperation(op)
	return nil
}

// UndoOperation undoes a specific operation by ID
func (s *Service) UndoOperation(opID string) error {
	var op Operation
	found := false
	for _, o := range s.history.Operations {
		if o.ID == opID {
			op = o
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("operation %s not found", opID)
	}

	// Handle different operation types
	switch op.Type {
	case "commit":
		return s.undoCommit(op)
	case "merge":
		return s.undoMerge(op)
	case "rebase":
		return s.undoRebase(op)
	default:
		return fmt.Errorf("unsupported operation type: %s", op.Type)
	}
}

// UndoLast undoes the last n operations
func (s *Service) UndoLast(n int) error {
	if n <= 0 {
		return fmt.Errorf("invalid number of operations to undo: %d", n)
	}

	if len(s.history.Operations) == 0 {
		return fmt.Errorf("no operations to undo")
	}

	if n > len(s.history.Operations) {
		n = len(s.history.Operations)
	}

	for i := 0; i < n; i++ {
		if err := s.UndoOperation(s.history.Operations[i].ID); err != nil {
			return fmt.Errorf("failed to undo operation %d: %w", i+1, err)
		}
	}

	return nil
}

// undoCommit handles undoing a commit operation
func (s *Service) undoCommit(op Operation) error {
	// Check if we need to handle stashed changes
	if op.Metadata.Stashed {
		defer s.git.StashPop()
	}

	// Reset to the previous commit
	if err := s.git.ResetSoft(op.Ref + "~1"); err != nil {
		return fmt.Errorf("failed to reset commit: %w", err)
	}

	return nil
}

// undoMerge handles undoing a merge operation
func (s *Service) undoMerge(op Operation) error {
	if merging, _ := s.git.IsMerging(); merging {
		return s.git.MergeAbort()
	}

	// If merge is completed, reset to pre-merge state
	return s.git.ResetSoft(op.Ref)
}

// undoRebase handles undoing a rebase operation
func (s *Service) undoRebase(op Operation) error {
	if rebasing, _ := s.git.IsRebasing(); rebasing {
		return s.git.RebaseAbort()
	}

	// If rebase is completed, reset to pre-rebase state
	return s.git.ResetSoft(op.Ref)
}

// GetHistory returns the undo history
func (s *Service) GetHistory() *History {
	return s.history
}

// LoadHistory loads the undo history from disk
func (s *Service) LoadHistory(repoPath string) error {
	return s.history.Load(repoPath)
}

// SaveHistory saves the undo history to disk
func (s *Service) SaveHistory(repoPath string) error {
	return s.history.Save(repoPath)
}
