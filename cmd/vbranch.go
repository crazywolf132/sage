package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/vbranch"
	"github.com/spf13/cobra"
)

var (
	vbranchManager vbranch.Manager
	vbranchWatcher *vbranch.Watcher
)

func initVirtualBranches() error {
	gitService := git.NewShellGit()

	var err error
	vbranchManager, err = vbranch.NewManager(gitService)
	if err != nil {
		return fmt.Errorf("failed to create virtual branch manager: %w", err)
	}

	vbranchWatcher, err = vbranch.NewWatcher(vbranchManager)
	if err != nil {
		return fmt.Errorf("failed to create virtual branch watcher: %w", err)
	}

	if err := vbranchWatcher.Start(); err != nil {
		return fmt.Errorf("failed to start virtual branch watcher: %w", err)
	}

	return nil
}

var vbranchCmd = &cobra.Command{
	Use:   "vbranch",
	Short: "Manage virtual branches",
	Long: `Virtual branches allow you to work on multiple features simultaneously
without switching Git branches. Changes are tracked separately and can be
materialized into real Git branches when ready.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := initVirtualBranches(); err != nil {
			return err
		}
		return nil
	},
}

var createVbranchCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new virtual branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		baseBranch, _ := cmd.Flags().GetString("base")

		if baseBranch == "" {
			gitService := git.NewShellGit()
			var err error
			baseBranch, err = gitService.CurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to get current branch: %w", err)
			}
		}

		vb, err := vbranchManager.CreateVirtualBranch(name, baseBranch)
		if err != nil {
			return fmt.Errorf("failed to create virtual branch: %w", err)
		}

		// Set this as the active branch in the watcher
		vbranchWatcher.SetActiveBranch(name)

		fmt.Printf("Created virtual branch '%s' based on '%s'\n", vb.Name, vb.BaseBranch)
		return nil
	},
}

var listVbranchCmd = &cobra.Command{
	Use:   "list",
	Short: "List all virtual branches",
	RunE: func(cmd *cobra.Command, args []string) error {
		branches, err := vbranchManager.ListVirtualBranches()
		if err != nil {
			return fmt.Errorf("failed to list virtual branches: %w", err)
		}

		if len(branches) == 0 {
			fmt.Println("No virtual branches found")
			return nil
		}

		for _, vb := range branches {
			status := " "
			if vb.Active {
				status = "*"
			}
			fmt.Printf("%s %-20s (based on %s, %d changes)\n", status, vb.Name, vb.BaseBranch, len(vb.Changes))
		}
		return nil
	},
}

var switchVbranchCmd = &cobra.Command{
	Use:   "switch [name]",
	Short: "Switch to a virtual branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := vbranchManager.ApplyVirtualBranch(name); err != nil {
			return fmt.Errorf("failed to switch to virtual branch: %w", err)
		}

		// Set this as the active branch in the watcher
		vbranchWatcher.SetActiveBranch(name)

		fmt.Printf("Switched to virtual branch '%s'\n", name)
		return nil
	},
}

var materializeVbranchCmd = &cobra.Command{
	Use:   "materialize [name]",
	Short: "Convert a virtual branch into a real Git branch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if err := vbranchManager.MaterializeBranch(name); err != nil {
			return fmt.Errorf("failed to materialize branch: %w", err)
		}

		// Clear the active branch in the watcher since it's now a real branch
		vbranchWatcher.SetActiveBranch("")

		fmt.Printf("Successfully materialized virtual branch '%s' into a real Git branch\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vbranchCmd)
	vbranchCmd.AddCommand(createVbranchCmd)
	vbranchCmd.AddCommand(listVbranchCmd)
	vbranchCmd.AddCommand(switchVbranchCmd)
	vbranchCmd.AddCommand(materializeVbranchCmd)

	createVbranchCmd.Flags().StringP("base", "b", "", "Base branch to create from (defaults to current branch)")
}
