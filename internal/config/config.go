package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// SandboxPolicy represents user-configurable sandbox restrictions
type SandboxPolicy struct {
	AllowReadPaths  []string `mapstructure:"allow_read_paths"`
	AllowWritePaths []string `mapstructure:"allow_write_paths"`
	BlockPaths      []string `mapstructure:"block_paths"`
	AllowNetwork    bool     `mapstructure:"allow_network"`
	AllowedHosts    []string `mapstructure:"allowed_hosts"`
	MaxProcesses    int      `mapstructure:"max_processes"`
	MaxMemoryMB     int      `mapstructure:"max_memory_mb"`
	MaxCPUTime      int      `mapstructure:"max_cpu_time"` // seconds
	AllowedCommands []string `mapstructure:"allowed_commands"`
	BlockedCommands []string `mapstructure:"blocked_commands"`
	AllowedEnvVars  []string `mapstructure:"allowed_env_vars"`
	BlockedEnvVars  []string `mapstructure:"blocked_env_vars"`
}

// Config represents the complete configuration for RubrDuck
type Config struct {
	Provider  string              `mapstructure:"provider"`
	Model     string              `mapstructure:"model"`
	Providers map[string]Provider `mapstructure:"providers"`
	Agent     AgentConfig         `mapstructure:"agent"`
	API       APIConfig           `mapstructure:"api"`
	History   HistoryConfig       `mapstructure:"history"`
	Sandbox   SandboxPolicy       `mapstructure:"sandbox"`
	// TUI holds configuration for the terminal user interface
	TUI     TUIConfig     `mapstructure:"tui"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// TUIConfig holds settings for the terminal UI modes
type TUIConfig struct {
	// StartMode sets the initial UI mode ("planning", "building", "debugging", "enhance").
	// If empty or unrecognized, the UI will prompt to select a mode.
	StartMode string `mapstructure:"start_mode"`

	// Timeouts for different modes (in seconds)
	PlanningTimeout int `mapstructure:"planning_timeout"`
	BuildingTimeout int `mapstructure:"building_timeout"`
	DebugTimeout    int `mapstructure:"debug_timeout"`
	EnhanceTimeout  int `mapstructure:"enhance_timeout"`
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
	Timeout        int    `mapstructure:"timeout"` // General timeout in seconds
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

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
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
	viper.SetDefault("agent.timeout", 300) // 5 minutes default

	// API defaults
	viper.SetDefault("api.enabled", false)
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("api.host", "localhost")

	// History defaults
	viper.SetDefault("history.max_size", 1000)
	viper.SetDefault("history.save_history", true)
	viper.SetDefault("history.sensitive_patterns", []string{})

	// Sandbox policy defaults
	viper.SetDefault("sandbox.allow_read_paths", []string{"./"})
	viper.SetDefault("sandbox.allow_write_paths", []string{"./"})
	viper.SetDefault("sandbox.block_paths", []string{"/etc", "/var", "/usr", "/bin", "/sbin", "/System"})
	viper.SetDefault("sandbox.allow_network", false)
	viper.SetDefault("sandbox.allowed_hosts", []string{})
	viper.SetDefault("sandbox.max_processes", 10)
	viper.SetDefault("sandbox.max_memory_mb", 512)
	viper.SetDefault("sandbox.max_cpu_time", 30)
	viper.SetDefault("sandbox.allowed_commands", []string{
		"ls", "cat", "head", "tail", "grep", "find", "wc", "sort", "uniq",
		"echo", "pwd", "whoami", "date", "ps", "git", "go", "npm", "yarn", "python", "node", "make",
	})
	viper.SetDefault("sandbox.blocked_commands", []string{
		"rm", "rmdir", "del", "format", "mkfs", "dd", "shred",
		"sudo", "su", "chmod", "chown", "passwd", "useradd",
		"wget", "curl", "nc", "netcat", "ssh", "scp", "rsync",
	})
	viper.SetDefault("sandbox.allowed_env_vars", []string{"PATH", "HOME", "USER", "PWD", "LANG", "LC_ALL"})
	viper.SetDefault("sandbox.blocked_env_vars", []string{"SUDO_ASKPASS", "SSH_AUTH_SOCK", "GPG_AGENT_INFO"})

	// TUI defaults
	viper.SetDefault("tui.start_mode", "")
	viper.SetDefault("tui.planning_timeout", 300) // 5 minutes for planning
	viper.SetDefault("tui.building_timeout", 180) // 3 minutes for building
	viper.SetDefault("tui.debug_timeout", 120)    // 2 minutes for debugging
	viper.SetDefault("tui.enhance_timeout", 120)  // 2 minutes for enhance

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "")
	viper.SetDefault("logging.max_size", 10)
	viper.SetDefault("logging.max_backups", 3)
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
