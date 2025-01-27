package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	explain bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = NewRootCmd()

// NewRootCmd creates a new instance of the root command
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sage",
		Short: "A Git helper for common workflows",
		Long: `Sage is a Git helper that provides shortcuts and automation for common Git workflows.
It aims to make Git operations more intuitive and faster.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cfgFile != "" {
				// Use config file from the flag
				viper.SetConfigFile(cfgFile)
			} else {
				// Find home directory
				home, err := os.UserHomeDir()
				cobra.CheckErr(err)

				// Search config in home directory with name ".sage" (without extension)
				viper.AddConfigPath(home)
				viper.SetConfigType("yaml")
				viper.SetConfigName(".sage")
			}

			// If a config file is found, read it in
			if err := viper.ReadInConfig(); err == nil {
				fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
			}
		},
	}

	// Add global flags
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Use a custom config file (default is $HOME/.sage.yaml)")
	cmd.PersistentFlags().BoolVar(&explain, "explain", false, "Show the underlying Git commands that Sage executes")

	// Add subcommands
	cmd.AddCommand(commitCmd)
	cmd.AddCommand(undoCmd)

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return RootCmd.Execute()
}
