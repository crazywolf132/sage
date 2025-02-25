package ui

import (
	"fmt"
	"strings"
	"time"
)

// SyncProgress represents a visual progress tracker for the sync operation
type SyncProgress struct {
	StartTime time.Time
	Steps     []SyncStep
	spinner   *Spinner
}

// SyncStep represents a single step in the sync operation
type SyncStep struct {
	Name        string
	Description string
	Status      string // "pending", "running", "success", "fail", "skip"
	StartTime   time.Time
	EndTime     time.Time
}

// NewSyncProgress creates a new sync progress tracker
func NewSyncProgress() *SyncProgress {
	return &SyncProgress{
		StartTime: time.Now(),
		Steps: []SyncStep{
			{Name: "verify", Description: "Repository verification", Status: "pending"},
			{Name: "stash", Description: "Work in progress", Status: "pending"},
			{Name: "fetch", Description: "Fetching updates", Status: "pending"},
			{Name: "integrate", Description: "Integrating changes", Status: "pending"},
			{Name: "push", Description: "Pushing changes", Status: "pending"},
			{Name: "restore", Description: "Restoring work", Status: "pending"},
		},
		spinner: NewSpinner(),
	}
}

// StartStep starts a step and updates the progress display
func (sp *SyncProgress) StartStep(stepName string) {
	for i := range sp.Steps {
		if sp.Steps[i].Name == stepName {
			sp.Steps[i].Status = "running"
			sp.Steps[i].StartTime = time.Now()
			
			// Update display
			sp.spinner.Start(sp.Steps[i].Description)
			return
		}
	}
}

// CompleteStep marks a step as completed and updates the progress display
func (sp *SyncProgress) CompleteStep(stepName string, success bool) {
	for i := range sp.Steps {
		if sp.Steps[i].Name == stepName {
			if success {
				sp.Steps[i].Status = "success"
			} else {
				sp.Steps[i].Status = "fail"
			}
			sp.Steps[i].EndTime = time.Now()
			
			// Update display
			if success {
				sp.spinner.StopSuccess()
			} else {
				sp.spinner.StopFail()
			}
			return
		}
	}
}

// SkipStep marks a step as skipped and updates the progress display
func (sp *SyncProgress) SkipStep(stepName string) {
	for i := range sp.Steps {
		if sp.Steps[i].Name == stepName {
			sp.Steps[i].Status = "skip"
			
			// Don't update spinner - we're skipping silently
			return
		}
	}
}

// GetSummary returns a summary of the sync operation
func (sp *SyncProgress) GetSummary() string {
	var sb strings.Builder
	
	// Calculate total time
	duration := time.Since(sp.StartTime).Round(time.Millisecond)
	
	sb.WriteString(fmt.Sprintf("\n%s Sync completed in %s\n\n", Bold("Summary:"), duration))
	
	// List steps with status
	for _, step := range sp.Steps {
		switch step.Status {
		case "success":
			sb.WriteString(fmt.Sprintf("%s %s\n", Green("✓"), step.Description))
		case "fail":
			sb.WriteString(fmt.Sprintf("%s %s\n", Red("✗"), step.Description))
		case "skip":
			sb.WriteString(fmt.Sprintf("%s %s\n", Gray("-"), step.Description))
		default:
			// Should not happen in a summary, but just in case
			sb.WriteString(fmt.Sprintf("%s %s\n", Yellow("?"), step.Description))
		}
	}
	
	return sb.String()
}

// ShowOptimizationTip displays a tip about enabling Git optimizations if sync was slow
func (sp *SyncProgress) ShowOptimizationTip() {
	// Only show tip if sync took more than 5 seconds
	duration := time.Since(sp.StartTime)
	if duration > 5*time.Second {
		fmt.Printf("\n%s Sync took %s. Enable optimizations with:\n", Yellow("Tip:"), Bold(duration.Round(time.Millisecond).String()))
		fmt.Printf("  sage config set experimental.commit-graph true\n")
		fmt.Printf("  sage config set experimental.fsmonitor true\n")
	}
} 