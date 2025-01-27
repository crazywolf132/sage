package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or modify Sage configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		viper.Set(key, value)

		// Attempt to save to the config file if possible
		if err := viper.WriteConfig(); err != nil {
			// If no config exists yet, try to write a new one
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				if err2 := viper.SafeWriteConfig(); err2 != nil {
					return fmt.Errorf("failed to write config: %w", err2)
				}
			} else {
				return fmt.Errorf("failed to write config: %w", err)
			}
		}

		fmt.Printf("Configuration updated: %s=%s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val := viper.Get(key)
		if val == nil {
			fmt.Printf("%s is not set\n", key)
		} else {
			fmt.Printf("%s=%v\n", key, val)
		}
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all Sage configuration settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := viper.AllSettings()
		if len(settings) == 0 {
			fmt.Println("No configuration found.")
		} else {
			for k, v := range settings {
				fmt.Printf("%s = %v\n", k, v)
			}
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configShowCmd)
}
