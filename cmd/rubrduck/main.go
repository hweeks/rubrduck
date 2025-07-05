package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hammie/rubrduck/cmd/rubrduck/commands"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration early to get logging settings
	cfg, err := config.Load()
	if err != nil {
		// If config loading fails, fall back to basic console logging
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		log.Warn().Err(err).Msg("Failed to load config, using default logging")
	} else {
		// Setup logging based on configuration
		setupLogging(cfg)
	}

	// Execute the root command
	if err := commands.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}

func setupLogging(cfg *config.Config) {
	// Get log level from config/environment
	globalLevel := getLogLevel(cfg)

	var writers []io.Writer

	// Console writer for immediate feedback - only INFO and above
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}

	// Add filtered console writer
	writers = append(writers, &ConsoleFilter{Writer: consoleWriter})

	// Add file writer if configured
	if logFile := getLogFilePath(cfg); logFile != "" {
		// Ensure log directory exists
		if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
			log.Warn().Err(err).Str("path", logFile).Msg("Failed to create log directory")
		} else {
			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				log.Warn().Err(err).Str("path", logFile).Msg("Failed to open log file")
			} else {
				// File gets all levels - no filtering
				writers = append(writers, file)
				log.Info().Str("log_file", logFile).Msg("Logging to file enabled")
			}
		}
	}

	// Configure multi-writer
	multi := io.MultiWriter(writers...)
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()

	// Set global log level
	zerolog.SetGlobalLevel(globalLevel)

	log.Info().
		Str("level", globalLevel.String()).
		Str("console_level", "info+").
		Int("writers", len(writers)).
		Msg("Logging initialized")
}

// ConsoleFilter filters out debug/trace logs from console output
type ConsoleFilter struct {
	Writer io.Writer
}

func (w *ConsoleFilter) Write(p []byte) (n int, err error) {
	// Check if this is a debug or trace log
	logStr := string(p)
	if strings.Contains(logStr, `"level":"debug"`) || strings.Contains(logStr, `"level":"trace"`) {
		// Don't write debug/trace to console
		return len(p), nil
	}

	// Write everything else to console
	return w.Writer.Write(p)
}

func getLogFilePath(cfg *config.Config) string {
	// Check if logging is configured in config
	logFile := cfg.Logging.File
	if logFile == "" {
		// Fall back to default location
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		logFile = filepath.Join(home, ".rubrduck", "rubrduck.log")
	}

	// Expand home directory if needed
	if strings.HasPrefix(logFile, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		logFile = filepath.Join(home, logFile[2:])
	}

	return logFile
}

func getLogLevel(cfg *config.Config) zerolog.Level {
	// Check environment first
	if os.Getenv("DEBUG") == "true" {
		return zerolog.DebugLevel
	}

	// Use config if available
	level := cfg.Logging.Level
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
