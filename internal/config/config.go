package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config represents the complete configuration for RubrDuck
type Config struct {
	Provider  string              `mapstructure:"provider"`
	Model     string              `mapstructure:"model"`
	Providers map[string]Provider `mapstructure:"providers"`
	Agent     AgentConfig         `mapstructure:"agent"`
	API       APIConfig           `mapstructure:"api"`
	History   HistoryConfig       `mapstructure:"history"`
}

// Provider represents an AI provider configuration
type Provider struct {
	Name    string `mapstructure:"name"`
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	EnvKey  string `mapstructure:"env_key"`
}

// AgentConfig represents agent-specific settings
type AgentConfig struct {
	ApprovalMode   string `mapstructure:"approval_mode"`
	SandboxEnabled bool   `mapstructure:"sandbox_enabled"`
	MaxRetries     int    `mapstructure:"max_retries"`
}

// APIConfig represents API server settings
type APIConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Port      int    `mapstructure:"port"`
	Host      string `mapstructure:"host"`
	AuthToken string `mapstructure:"auth_token"`
}

// HistoryConfig represents conversation history settings
type HistoryConfig struct {
	MaxSize           int      `mapstructure:"max_size"`
	SaveHistory       bool     `mapstructure:"save_history"`
	SensitivePatterns []string `mapstructure:"sensitive_patterns"`
}

// Load loads the configuration from viper
func Load() (*Config, error) {
	// Set defaults
	setDefaults()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Load API keys from environment variables
	for providerName, provider := range cfg.Providers {
		if provider.EnvKey != "" {
			if apiKey := os.Getenv(provider.EnvKey); apiKey != "" {
				provider.APIKey = apiKey
				cfg.Providers[providerName] = provider
			}
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Provider defaults
	viper.SetDefault("provider", "openai")
	viper.SetDefault("model", "gpt-4")

	// Default providers
	viper.SetDefault("providers.openai.name", "OpenAI")
	viper.SetDefault("providers.openai.base_url", "https://api.openai.com/v1")
	viper.SetDefault("providers.openai.env_key", "OPENAI_API_KEY")

	viper.SetDefault("providers.azure.name", "Azure OpenAI")
	viper.SetDefault("providers.azure.env_key", "AZURE_API_KEY")

	viper.SetDefault("providers.anthropic.name", "Anthropic")
	viper.SetDefault("providers.anthropic.base_url", "https://api.anthropic.com/v1")
	viper.SetDefault("providers.anthropic.env_key", "ANTHROPIC_API_KEY")

	// Agent defaults
	viper.SetDefault("agent.approval_mode", "suggest")
	viper.SetDefault("agent.sandbox_enabled", true)
	viper.SetDefault("agent.max_retries", 3)

	// API defaults
	viper.SetDefault("api.enabled", false)
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.host", "localhost")

	// History defaults
	viper.SetDefault("history.max_size", 1000)
	viper.SetDefault("history.save_history", true)
	viper.SetDefault("history.sensitive_patterns", []string{})
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate approval mode
	validModes := map[string]bool{
		"suggest":   true,
		"auto-edit": true,
		"full-auto": true,
	}
	if !validModes[c.Agent.ApprovalMode] {
		return fmt.Errorf("invalid approval mode: %s", c.Agent.ApprovalMode)
	}

	// Validate provider exists
	if _, ok := c.Providers[c.Provider]; !ok {
		return fmt.Errorf("provider %s not configured", c.Provider)
	}

	// Validate API key is available for selected provider
	provider := c.Providers[c.Provider]
	if provider.APIKey == "" && provider.EnvKey != "" {
		if os.Getenv(provider.EnvKey) == "" {
			return fmt.Errorf("API key not found for provider %s. Please set %s environment variable", c.Provider, provider.EnvKey)
		}
	}

	return nil
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".rubrduck"), nil
}

// EnsureConfigDir creates the configuration directory if it doesn't exist
func EnsureConfigDir() error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0755)
}
