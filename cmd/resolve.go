package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
)

// resolveCmd represents the resolve command
var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Interactively resolve merge conflicts",
	Long: `Sage resolve helps you handle merge conflicts interactively.

When you encounter conflicts during a merge, rebase, or sync operation,
this command will show you all files with conflicts and help you
resolve them by opening them in your preferred editor.

You can specify an editor with the --editor flag, or it will use your
Git-configured editor, or fall back to the EDITOR environment variable.`,
	Example: `  sage resolve               # List and resolve all conflicts
  sage resolve --editor vim  # Use vim to edit conflict files
  sage resolve --auto        # Attempt to auto-resolve simple conflicts`,
	RunE: func(cmd *cobra.Command, args []string) error {
		g := git.NewShellGit()

		autoResolve, _ := cmd.Flags().GetBool("auto")
		editorCmd, _ := cmd.Flags().GetString("editor")

		return resolveConflicts(g, editorCmd, autoResolve)
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().StringP("editor", "e", "", "Specify which editor to use for conflict resolution")
	resolveCmd.Flags().BoolP("auto", "a", false, "Attempt to automatically resolve simple conflicts")
}

// resolveConflicts handles the conflict resolution process
func resolveConflicts(g git.Service, editorCmd string, autoResolve bool) error {
	// Check if we're in a merge or rebase state
	isMerging, err := g.IsMerging()
	if err != nil {
		return err
	}

	isRebasing, err := g.IsRebasing()
	if err != nil {
		return err
	}

	if !isMerging && !isRebasing {
		return fmt.Errorf("No merge or rebase in progress. Nothing to resolve")
	}

	// Get conflicted files
	conflicts, err := g.ListConflictedFiles()
	if err != nil {
		return fmt.Errorf("Failed to list conflicted files: %w", err)
	}

	if conflicts == "" {
		ui.Success("No conflicts detected. Proceed with 'sage sync --continue'")
		return nil
	}

	conflictFiles := strings.Split(strings.TrimSpace(conflicts), "\n")

	if len(conflictFiles) == 0 || (len(conflictFiles) == 1 && conflictFiles[0] == "") {
		ui.Success("No conflicts detected. Proceed with 'sage sync --continue'")
		return nil
	}

	ui.Info(fmt.Sprintf("Found %d file(s) with conflicts:", len(conflictFiles)))
	for i, file := range conflictFiles {
		fmt.Printf("%d: %s\n", i+1, file)
	}

	// If auto-resolve flag is set, try that first
	if autoResolve {
		resolvedCount := 0
		spinner := ui.NewSpinner()
		spinner.Start("Attempting to auto-resolve conflicts...")

		// Try to resolve each conflict automatically
		for _, file := range conflictFiles {
			// This is a simple example that always takes "ours" version
			// In a real implementation, you might use git-merge-file or a more sophisticated algorithm
			if err := attemptAutoResolve(g, file); err == nil {
				resolvedCount++
			}
		}

		if resolvedCount > 0 {
			spinner.StopSuccess()
			ui.Success(fmt.Sprintf("Auto-resolved %d of %d conflict(s)", resolvedCount, len(conflictFiles)))

			// Get updated conflicts list
			conflicts, _ = g.ListConflictedFiles()
			if conflicts == "" {
				ui.Success("All conflicts resolved successfully!")
				return continueOperation(g, isMerging, isRebasing)
			}

			conflictFiles = strings.Split(strings.TrimSpace(conflicts), "\n")
			if len(conflictFiles) == 0 || (len(conflictFiles) == 1 && conflictFiles[0] == "") {
				ui.Success("All conflicts resolved successfully!")
				return continueOperation(g, isMerging, isRebasing)
			}

			ui.Info(fmt.Sprintf("Still %d file(s) with conflicts:", len(conflictFiles)))
			for i, file := range conflictFiles {
				fmt.Printf("%d: %s\n", i+1, file)
			}
		} else {
			spinner.StopFail()
			ui.Info("Could not auto-resolve any conflicts. Manual resolution required.")
		}
	}

	// Determine which editor to use
	editor := determineEditor(g, editorCmd)

	// Interactive resolution
	ui.Info("Enter the number of the file to edit (or 'a' for all, 'c' to continue if done, 'q' to quit):")

	var input string
	for {
		fmt.Print("> ")
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "q", "quit", "exit":
			return nil
		case "c", "continue":
			// Check if there are still conflicts
			remainingConflicts, _ := g.ListConflictedFiles()
			if remainingConflicts != "" && remainingConflicts != "\n" {
				ui.Warning("There are still unresolved conflicts. Resolve all conflicts before continuing")
				continue
			}
			return continueOperation(g, isMerging, isRebasing)
		case "a", "all":
			// Open all files
			for _, file := range conflictFiles {
				if err := openFileInEditor(editor, file); err != nil {
					ui.Error(fmt.Sprintf("Failed to open %s: %v", file, err))
				}
			}
		default:
			// Try to parse as a number
			var fileIndex int
			if _, err := fmt.Sscanf(input, "%d", &fileIndex); err == nil {
				fileIndex-- // Convert to 0-based index
				if fileIndex >= 0 && fileIndex < len(conflictFiles) {
					if err := openFileInEditor(editor, conflictFiles[fileIndex]); err != nil {
						ui.Error(fmt.Sprintf("Failed to open %s: %v", conflictFiles[fileIndex], err))
					}
				} else {
					ui.Warning(fmt.Sprintf("Invalid file number. Please enter 1-%d", len(conflictFiles)))
				}
			} else {
				ui.Warning("Invalid input. Enter a file number, 'a' for all, 'c' to continue, or 'q' to quit")
			}
		}

		// After each action, refresh the conflict list
		conflicts, _ = g.ListConflictedFiles()
		if conflicts == "" {
			ui.Success("All conflicts resolved!")
			ui.Info("Enter 'c' to continue or 'q' to quit without continuing")
			continue
		}

		newConflictFiles := strings.Split(strings.TrimSpace(conflicts), "\n")
		if len(newConflictFiles) == 0 || (len(newConflictFiles) == 1 && newConflictFiles[0] == "") {
			ui.Success("All conflicts resolved!")
			ui.Info("Enter 'c' to continue or 'q' to quit without continuing")
			continue
		}

		// Only update the list if there are actual changes
		if !equalStringSlices(conflictFiles, newConflictFiles) {
			conflictFiles = newConflictFiles
			ui.Info(fmt.Sprintf("Remaining files with conflicts (%d):", len(conflictFiles)))
			for i, file := range conflictFiles {
				fmt.Printf("%d: %s\n", i+1, file)
			}
		}
	}
}

// determineEditor gets the editor to use based on priority:
// 1. Command line flag
// 2. Git config core.editor
// 3. EDITOR environment variable
// 4. Default (vim)
func determineEditor(g git.Service, editorFlag string) string {
	if editorFlag != "" {
		return editorFlag
	}

	// Try to get from git config
	gitEditor, err := g.GetConfigValue("core.editor")
	if err == nil && gitEditor != "" {
		return gitEditor
	}

	// Try environment variable
	envEditor := os.Getenv("EDITOR")
	if envEditor != "" {
		return envEditor
	}

	// Default
	return "vim"
}

// openFileInEditor opens the specified file in the given editor
func openFileInEditor(editor, filePath string) error {
	ui.Info(fmt.Sprintf("Opening %s with %s...", filePath, editor))

	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// attemptAutoResolve tries to automatically resolve conflicts in a file
// This is a simplistic implementation that could be enhanced
func attemptAutoResolve(g git.Service, filePath string) error {
	// This is a placeholder for auto-resolution logic
	// A real implementation might use git-merge-file with various strategies
	// or parse the conflict markers and apply heuristics

	// Example: always take "ours" version
	// In practice, you might want more sophisticated conflict resolution
	// return g.run("checkout", "--ours", filePath)

	// For now, return an error to indicate we couldn't auto-resolve
	return fmt.Errorf("auto-resolution not implemented")
}

// continueOperation continues the current merge/rebase operation
func continueOperation(g git.Service, isMerging, isRebasing bool) error {
	if isMerging {
		ui.Info("Continuing merge...")
		return g.MergeContinue()
	} else if isRebasing {
		ui.Info("Continuing rebase...")
		return g.RebaseContinue()
	}
	return nil
}

// equalStringSlices checks if two string slices have the same elements
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
