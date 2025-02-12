package ui

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

// Spinner provides a progress indicator
type Spinner struct {
	s *spinner.Spinner
}

// NewSpinner creates a new spinner with default settings
func NewSpinner() *Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Color("cyan")
	return &Spinner{s: s}
}

// Start begins the spinner animation with a message
func (s *Spinner) Start(message string) {
	s.s.Suffix = fmt.Sprintf(" %s", message)
	s.s.Start()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.s.Stop()
}

// StopSuccess stops the spinner with a success message
func (s *Spinner) StopSuccess() {
	s.s.Stop()
	fmt.Printf("%s %s\n", Green("✓"), s.s.Suffix)
}

// StopFail stops the spinner with a failure message
func (s *Spinner) StopFail() {
	s.s.Stop()
	fmt.Printf("%s %s\n", Red("✗"), s.s.Suffix)
}
