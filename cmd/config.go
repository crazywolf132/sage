package cmd

import (
	"fmt"

	"github.com/crazywolf132/sage/internal/config"
	"github.com/crazywolf132/sage/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Sage configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Args:  cobra.ExactArgs(1),
	Short: "Get a config value",
	RunE: func(cmd *cobra.Command, args []string) error {
		val := config.Get(args[0])
		if val == "" {
			fmt.Println(ui.Gray("not set"))
		} else {
			fmt.Println(val)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Args:  cobra.ExactArgs(2),
	Short: "Set a config value (by default in local config if in a repo, otherwise global)",
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		err := config.Set(key, value)
		if err != nil {
			return err
		}
		fmt.Printf("%s %s=%s\n", ui.Green("Set"), key, value)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}
