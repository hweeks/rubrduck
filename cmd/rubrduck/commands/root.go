package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/hammie/rubrduck/internal/config"
	"github.com/hammie/rubrduck/internal/tui"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile      string
	provider     string
	model        string
	approvalMode string
)

// tuiSnapshotsCmd generates static text snapshots for each TUI mode
var tuiSnapshotsCmd = &cobra.Command{
	Use:   "tui-snapshots",
	Short: "Generate static snapshots of all TUI modes",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		for _, mode := range tui.ModeOptions {
			m := tui.NewModel(cfg)
			m.SetMode(mode)
			m.SetSize(80, 24)
			m.SetFocused(true)
			fmt.Printf("=== %s ===\n%s\n\n", tui.ModeName(mode), m.View())
		}
		return nil
	},
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rubrduck",
	Short: "AI-powered coding agent for IDEs",
	Long: `RubrDuck is a CLI tool that brings AI-assisted coding to your terminal and IDE.
	
It provides an interactive TUI for chatting with AI models, executing code,
and managing your development workflow with built-in safety features.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If arguments are provided, treat them as a prompt
		if len(args) > 0 {
			prompt := strings.Join(args, " ")
			return runWithPrompt(prompt)
		}

		// Otherwise, start interactive TUI
		return runInteractiveTUI()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.rubrduck/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&provider, "provider", "openai", "AI provider to use")
	rootCmd.PersistentFlags().StringVar(&model, "model", "", "AI model to use (provider-specific)")
	rootCmd.PersistentFlags().StringVarP(&approvalMode, "approval-mode", "a", "suggest", "Approval mode: suggest, auto-edit, or full-auto")

	// Bind flags to viper
   _ = viper.BindPFlag("provider", rootCmd.PersistentFlags().Lookup("provider"))
   _ = viper.BindPFlag("model", rootCmd.PersistentFlags().Lookup("model"))
   _ = viper.BindPFlag("agent.approval_mode", rootCmd.PersistentFlags().Lookup("approval-mode"))

	// Add subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	// Generate static snapshots of the TUI modes
	rootCmd.AddCommand(tuiSnapshotsCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error().Err(err).Msg("Failed to get home directory")
			os.Exit(1)
		}

		// Search config in home directory with name ".rubrduck" (without extension).
		configPath := fmt.Sprintf("%s/.rubrduck", home)
		viper.AddConfigPath(configPath)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")

		// Create config directory if it doesn't exist
		if err := os.MkdirAll(configPath, 0755); err != nil {
			log.Error().Err(err).Msg("Failed to create config directory")
		}
	}

	viper.SetEnvPrefix("RUBRDUCK")
	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Debug().Str("config", viper.ConfigFileUsed()).Msg("Using config file")
	} else {
		log.Debug().Msg("No config file found, using defaults and environment variables")
	}
}

func runWithPrompt(prompt string) error {
	log.Info().Str("prompt", prompt).Msg("Running with prompt")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// TODO: Implement agent execution with prompt
	fmt.Printf("Running with prompt: %s\n", prompt)
	fmt.Printf("Provider: %s, Model: %s, Approval Mode: %s\n",
		cfg.Provider, cfg.Model, cfg.Agent.ApprovalMode)

	return nil
}

func runInteractiveTUI() error {
	log.Info().Msg("Starting interactive TUI")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Start the TUI
	return tui.Run(cfg)
}
