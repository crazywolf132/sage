package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

// Spinner provides a progress indicator
type Spinner struct {
	spinner *spinner.Spinner
}

// NewSpinner creates a new spinner instance
func NewSpinner() *Spinner {
	s := &Spinner{}
	s.spinner = spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.spinner.Suffix = " " // Add a space after the spinner
	return s
}

// Start starts the spinner with the given message
func (s *Spinner) Start(msg string) {
	s.spinner.Suffix = " " + msg // Ensure space between spinner and message
	s.spinner.Start()
}

// Stop stops the spinner without any symbol
func (s *Spinner) Stop() {
	s.spinner.Stop()
}

// StopSuccess stops the spinner with a success symbol
func (s *Spinner) StopSuccess() {
	s.spinner.Stop()
	fmt.Printf("✓ %s\n", strings.TrimSpace(s.spinner.Suffix)) // Ensure consistent spacing
}

// StopFail stops the spinner with a failure symbol
func (s *Spinner) StopFail() {
	s.spinner.Stop()
	fmt.Printf("✗ %s\n", strings.TrimSpace(s.spinner.Suffix)) // Ensure consistent spacing
}
