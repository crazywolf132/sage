package app

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/undo"
)

// Undo handles undoing Git operations
func Undo(g git.Service, count int) error {
	s := undo.NewService(g)
	if err := s.LoadHistory("."); err != nil {
		return fmt.Errorf("failed to load undo history: %w", err)
	}

	return s.UndoLast(count)
}

// RecordOperation records a Git operation in the undo history
func RecordOperation(g git.Service, opType, description, command, category string, files []string, branch string, message string, stashed bool, stashRef string) error {
	s := undo.NewService(g)
	if err := s.LoadHistory("."); err != nil {
		return fmt.Errorf("failed to load undo history: %w", err)
	}

	// Create metadata for the operation
	op := undo.Operation{
		Type:        opType,
		Description: description,
		Command:     command,
		Category:    category,
	}
	op.Metadata.Files = files
	op.Metadata.Branch = branch
	op.Metadata.Message = message
	op.Metadata.Stashed = stashed
	op.Metadata.StashRef = stashRef

	if err := s.RecordOperation(opType, description, command, category, op); err != nil {
		return fmt.Errorf("failed to record operation: %w", err)
	}

	return s.SaveHistory(".")
}
