package main

import (
	"os"

	"github.com/hammie/rubrduck/cmd/rubrduck/commands"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Set log level based on environment
	if os.Getenv("DEBUG") == "true" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Execute the root command
	if err := commands.Execute(); err != nil {
		log.Fatal().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
