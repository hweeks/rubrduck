package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage RubrDuck configuration",
	Long:  `Display and manage RubrDuck configuration settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := viper.AllSettings()
		for key, value := range settings {
			fmt.Printf("%s: %v\n", key, value)
		}
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	RunE: func(cmd *cobra.Command, args []string) error {
		if configFile := viper.ConfigFileUsed(); configFile != "" {
			fmt.Println(configFile)
		} else {
			fmt.Println("No configuration file found")
		}
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
}
