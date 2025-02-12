package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Run the virtual branch daemon",
	Hidden: true, // Hide from help as it's meant to be run internally
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := initVirtualBranches(); err != nil {
			return fmt.Errorf("failed to initialize virtual branches: %w", err)
		}

		// Create pid file
		pidFile := filepath.Join(os.TempDir(), "sage-vbranch.pid")
		if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
			return fmt.Errorf("failed to create pid file: %w", err)
		}
		defer os.Remove(pidFile)

		// Setup signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		ui.Success("Virtual branch daemon is running")
		ui.Info("Monitoring file changes...")

		// Wait for signal
		<-sigChan
		ui.Info("Shutting down daemon...")
		vbranchWatcher.Stop()
		return nil
	},
}

func init() {
	vbranchCmd.AddCommand(daemonCmd)
}
