package vbranch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/crazywolf132/sage/internal/git"
)

type manager struct {
	git         git.Service
	stateDir    string // Path to .git/sage directory
	branches    map[string]*VirtualBranch
	activeCount int
}

// NewManager creates a new virtual branch manager
func NewManager(git git.Service) (Manager, error) {
	// Find .git directory
	gitDir, err := findGitDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find .git directory: %w", err)
	}

	// Create .git/sage directory if it doesn't exist
	stateDir := filepath.Join(gitDir, "sage")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	m := &manager{
		git:      git,
		stateDir: stateDir,
		branches: make(map[string]*VirtualBranch),
	}

	if err := m.loadState(); err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	return m, nil
}

// findGitDir walks up the directory tree to find the .git directory
func findGitDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(dir, ".git")
		if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
			return gitDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not a git repository (or any parent up to root)")
		}
		dir = parent
	}
}

func (m *manager) CreateVirtualBranch(name string, baseBranch string) (*VirtualBranch, error) {
	if _, exists := m.branches[name]; exists {
		return nil, fmt.Errorf("virtual branch %s already exists", name)
	}

	vb := &VirtualBranch{
		Name:        name,
		BaseBranch:  baseBranch,
		Created:     time.Now(),
		LastUpdated: time.Now(),
		Changes:     make([]Change, 0),
		Active:      false,
	}

	m.branches[name] = vb
	if err := m.saveState(); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	return vb, nil
}

func (m *manager) ListVirtualBranches() ([]*VirtualBranch, error) {
	branches := make([]*VirtualBranch, 0, len(m.branches))
	for _, vb := range m.branches {
		branches = append(branches, vb)
	}
	return branches, nil
}

func (m *manager) GetVirtualBranch(name string) (*VirtualBranch, error) {
	vb, exists := m.branches[name]
	if !exists {
		return nil, fmt.Errorf("virtual branch %s not found", name)
	}
	return vb, nil
}

func (m *manager) ApplyVirtualBranch(name string) error {
	vb, exists := m.branches[name]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", name)
	}

	if vb.Active {
		return nil // Already applied
	}

	// Check if working directory is clean
	clean, err := m.git.IsClean()
	if err != nil {
		return fmt.Errorf("failed to check if working directory is clean: %w", err)
	}

	// If working directory is not clean, stash changes in the current active branch
	if !clean {
		currentBranch := m.getActiveBranch()
		if currentBranch != nil {
			if err := m.stashChanges(currentBranch); err != nil {
				return fmt.Errorf("failed to stash changes: %w", err)
			}
		} else {
			return fmt.Errorf("working directory is not clean and no active virtual branch found")
		}
	}

	// Create a temporary patch file
	patchFile := filepath.Join(m.stateDir, fmt.Sprintf("%s.patch", name))
	if err := m.createPatchFile(vb, patchFile); err != nil {
		return fmt.Errorf("failed to create patch file: %w", err)
	}
	defer os.Remove(patchFile)

	// Apply the patch using git apply
	if err := m.git.RunInteractive("apply", "--3way", patchFile); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	// If this branch has stashed changes, apply them
	if vb.StashedDiff != "" {
		if err := m.applyStashedChanges(vb); err != nil {
			return fmt.Errorf("failed to apply stashed changes: %w", err)
		}
	}

	vb.Active = true
	m.activeCount++

	return m.saveState()
}

func (m *manager) UnapplyVirtualBranch(name string) error {
	vb, exists := m.branches[name]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", name)
	}

	if !vb.Active {
		return nil // Already unapplied
	}

	// Check for uncommitted changes
	clean, err := m.git.IsClean()
	if err != nil {
		return fmt.Errorf("failed to check if working directory is clean: %w", err)
	}

	// If there are changes, stash them in the virtual branch
	if !clean {
		if err := m.stashChanges(vb); err != nil {
			return fmt.Errorf("failed to stash changes: %w", err)
		}
	}

	// Create a temporary patch file
	patchFile := filepath.Join(m.stateDir, fmt.Sprintf("%s.patch", name))
	if err := m.createPatchFile(vb, patchFile); err != nil {
		return fmt.Errorf("failed to create patch file: %w", err)
	}
	defer os.Remove(patchFile)

	// Reverse apply the patch using git apply
	if err := m.git.RunInteractive("apply", "--reverse", "--3way", patchFile); err != nil {
		return fmt.Errorf("failed to unapply changes: %w", err)
	}

	vb.Active = false
	m.activeCount--

	return m.saveState()
}

func (m *manager) stashChanges(vb *VirtualBranch) error {
	// Get the current diff
	diff, err := m.git.GetDiff()
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	// Store the diff in the virtual branch
	vb.StashedDiff = diff

	// Reset the working directory
	if err := m.git.ResetSoft("HEAD"); err != nil {
		return fmt.Errorf("failed to reset working directory: %w", err)
	}

	return nil
}

func (m *manager) applyStashedChanges(vb *VirtualBranch) error {
	// Create a temporary patch file for stashed changes
	stashFile := filepath.Join(m.stateDir, fmt.Sprintf("%s.stash", vb.Name))
	if err := os.WriteFile(stashFile, []byte(vb.StashedDiff), 0644); err != nil {
		return fmt.Errorf("failed to write stash file: %w", err)
	}
	defer os.Remove(stashFile)

	// Apply the stashed changes
	if err := m.git.RunInteractive("apply", stashFile); err != nil {
		return fmt.Errorf("failed to apply stashed changes: %w", err)
	}

	// Clear the stashed diff
	vb.StashedDiff = ""
	return nil
}

func (m *manager) getActiveBranch() *VirtualBranch {
	for _, vb := range m.branches {
		if vb.Active {
			return vb
		}
	}
	return nil
}

func (m *manager) AddChange(branchName string, change Change) error {
	vb, exists := m.branches[branchName]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", branchName)
	}

	vb.Changes = append(vb.Changes, change)
	vb.LastUpdated = time.Now()

	return m.saveState()
}

func (m *manager) RemoveChange(branchName string, path string) error {
	vb, exists := m.branches[branchName]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", branchName)
	}

	for i, change := range vb.Changes {
		if change.Path == path {
			vb.Changes = append(vb.Changes[:i], vb.Changes[i+1:]...)
			vb.LastUpdated = time.Now()
			return m.saveState()
		}
	}

	return fmt.Errorf("change for path %s not found in branch %s", path, branchName)
}

func (m *manager) createPatchFile(vb *VirtualBranch, patchFile string) error {
	// Create a unified diff format patch
	var patch string
	for _, change := range vb.Changes {
		patch += fmt.Sprintf("diff --git a/%s b/%s\n", change.Path, change.Path)
		patch += fmt.Sprintf("--- a/%s\n", change.Path)
		patch += fmt.Sprintf("+++ b/%s\n", change.Path)
		patch += change.Diff + "\n"
	}

	return os.WriteFile(patchFile, []byte(patch), 0644)
}

func (m *manager) MaterializeBranch(name string) error {
	vb, exists := m.branches[name]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", name)
	}

	// Create actual Git branch
	if err := m.git.CreateBranch(name); err != nil {
		return fmt.Errorf("failed to create Git branch: %w", err)
	}

	// Apply changes and commit them
	if err := m.ApplyVirtualBranch(name); err != nil {
		return fmt.Errorf("failed to apply changes: %w", err)
	}

	// Stage and commit all changes
	if err := m.git.StageAll(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	commitMsg := fmt.Sprintf("Materialized virtual branch %s", vb.Name)
	if err := m.git.Commit(commitMsg, false, false); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	delete(m.branches, name)
	return m.saveState()
}

func (m *manager) MoveChanges(fromBranch string, toBranch string, paths []string) error {
	from, exists := m.branches[fromBranch]
	if !exists {
		return fmt.Errorf("source branch %s not found", fromBranch)
	}

	to, exists := m.branches[toBranch]
	if !exists {
		return fmt.Errorf("target branch %s not found", toBranch)
	}

	// Move specified changes between branches
	for _, path := range paths {
		for i, change := range from.Changes {
			if change.Path == path {
				to.Changes = append(to.Changes, change)
				from.Changes = append(from.Changes[:i], from.Changes[i+1:]...)
				break
			}
		}
	}

	from.LastUpdated = time.Now()
	to.LastUpdated = time.Now()

	return m.saveState()
}

func (m *manager) loadState() error {
	stateFile := filepath.Join(m.stateDir, "vbranches.json")
	data, err := os.ReadFile(stateFile)
	if os.IsNotExist(err) {
		m.branches = make(map[string]*VirtualBranch)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, &m.branches); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Count active branches
	m.activeCount = 0
	for _, vb := range m.branches {
		if vb.Active {
			m.activeCount++
		}
	}

	return nil
}

func (m *manager) saveState() error {
	data, err := json.MarshalIndent(m.branches, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	stateFile := filepath.Join(m.stateDir, "vbranches.json")
	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (m *manager) GetActiveBranch() (*VirtualBranch, error) {
	vb := m.getActiveBranch()
	if vb == nil {
		return nil, fmt.Errorf("no active virtual branch found")
	}
	return vb, nil
}

func (m *manager) HasStashedChanges(name string) (bool, error) {
	vb, exists := m.branches[name]
	if !exists {
		return false, fmt.Errorf("virtual branch %s not found", name)
	}
	return vb.StashedDiff != "", nil
}

func (m *manager) PopStashedChanges(name string) error {
	vb, exists := m.branches[name]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", name)
	}

	if vb.StashedDiff == "" {
		return fmt.Errorf("no stashed changes found in branch %s", name)
	}

	if !vb.Active {
		return fmt.Errorf("cannot pop stashed changes: branch %s is not active", name)
	}

	// Apply the stashed changes
	if err := m.applyStashedChanges(vb); err != nil {
		return fmt.Errorf("failed to apply stashed changes: %w", err)
	}

	return m.saveState()
}

func (m *manager) DropStashedChanges(name string) error {
	vb, exists := m.branches[name]
	if !exists {
		return fmt.Errorf("virtual branch %s not found", name)
	}

	if vb.StashedDiff == "" {
		return fmt.Errorf("no stashed changes found in branch %s", name)
	}

	// Clear the stashed changes
	vb.StashedDiff = ""
	return m.saveState()
}
