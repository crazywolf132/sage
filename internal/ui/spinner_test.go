package ui

import (
	"testing"
	"time"
)

func TestSpinner(t *testing.T) {
	// Test spinner creation
	t.Run("NewSpinner", func(t *testing.T) {
		s := NewSpinner()
		if s == nil {
			t.Error("Expected non-nil spinner")
		}
		if s.s == nil {
			t.Error("Expected non-nil internal spinner")
		}
	})

	// Test spinner lifecycle
	t.Run("SpinnerLifecycle", func(t *testing.T) {
		s := NewSpinner()
		message := "Testing spinner"

		// Test start
		s.Start(message)
		if s.s.Suffix != " "+message {
			t.Errorf("Expected spinner message %q, got %q", " "+message, s.s.Suffix)
		}

		// Let it spin briefly
		time.Sleep(200 * time.Millisecond)

		// Test stop
		s.Stop()
		// Can't test much after stop since it's visual output
	})

	// Test success stop
	t.Run("StopSuccess", func(t *testing.T) {
		s := NewSpinner()
		message := "Operation complete"
		s.Start(message)
		time.Sleep(100 * time.Millisecond)
		s.StopSuccess()
		// Visual output can't be easily tested
	})

	// Test failure stop
	t.Run("StopFail", func(t *testing.T) {
		s := NewSpinner()
		message := "Operation failed"
		s.Start(message)
		time.Sleep(100 * time.Millisecond)
		s.StopFail()
		// Visual output can't be easily tested
	})
}
