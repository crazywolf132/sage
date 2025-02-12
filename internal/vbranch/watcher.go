package vbranch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	manager      Manager
	watcher      *fsnotify.Watcher
	activeBranch string
	mu           sync.RWMutex
	done         chan struct{}
}

func NewWatcher(manager Manager) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	w := &Watcher{
		manager: manager,
		watcher: fsWatcher,
		done:    make(chan struct{}),
	}

	return w, nil
}

func (w *Watcher) Start() error {
	// Get current working directory
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk through all directories and add them to the watcher
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory and other ignored paths
		if info.IsDir() && shouldIgnore(path) {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return w.watcher.Add(path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	go w.watch()
	return nil
}

func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
}

func (w *Watcher) SetActiveBranch(name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.activeBranch = name
}

func (w *Watcher) watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Skip certain files
			if shouldIgnore(event.Name) {
				continue
			}

			w.handleFileEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("error: %v\n", err)

		case <-w.done:
			return
		}
	}
}

func (w *Watcher) handleFileEvent(event fsnotify.Event) {
	w.mu.RLock()
	activeBranch := w.activeBranch
	w.mu.RUnlock()

	if activeBranch == "" {
		return // No active virtual branch
	}

	// Only handle write and remove operations
	if event.Op&(fsnotify.Write|fsnotify.Remove) == 0 {
		return
	}

	// Get the relative path
	absPath, err := filepath.Abs(event.Name)
	if err != nil {
		fmt.Printf("error getting absolute path: %v\n", err)
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("error getting working directory: %v\n", err)
		return
	}

	relPath, err := filepath.Rel(wd, absPath)
	if err != nil {
		fmt.Printf("error getting relative path: %v\n", err)
		return
	}

	// Create a change record
	change := Change{
		Path:      relPath,
		Timestamp: time.Now(),
		Staged:    false,
	}

	// Get the file diff if it exists
	if event.Op&fsnotify.Write == fsnotify.Write {
		diff, err := generateDiff(relPath)
		if err != nil {
			fmt.Printf("error getting diff: %v\n", err)
			return
		}
		change.Diff = diff
	}

	// Add the change to the active branch
	if err := w.manager.AddChange(activeBranch, change); err != nil {
		fmt.Printf("error adding change: %v\n", err)
	}
}

func shouldIgnore(path string) bool {
	// Add patterns to ignore
	patterns := []string{
		".git",
		"node_modules",
		".DS_Store",
		"*.swp",
		"*.swo",
		"*.pyc",
		"__pycache__",
		".idea",
		".vscode",
		"*.log",
	}

	base := filepath.Base(path)
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, base)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// generateDiff creates a diff for the given file by comparing it with its HEAD version
func generateDiff(path string) (string, error) {
	// First check if the file is tracked by Git
	cmd := exec.Command("git", "ls-files", "--error-unmatch", path)
	if err := cmd.Run(); err != nil {
		// File is untracked, generate diff against empty file
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		return fmt.Sprintf("diff --git a/%s b/%s\n--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n%s",
			path, path, path, len(content), string(content)), nil
	}

	// File is tracked, generate diff against HEAD
	cmd = exec.Command("git", "diff", "--no-index", "--no-prefix", "HEAD", path)
	output, err := cmd.Output()
	if err != nil {
		// git diff returns exit code 1 if there are differences
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			return "", fmt.Errorf("failed to generate diff: %w", err)
		}
	}

	return string(output), nil
}
