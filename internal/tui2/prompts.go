package tui2

import (
	"sync"

	"github.com/hammie/rubrduck/internal/config"
	"github.com/hammie/rubrduck/internal/prompts"
	"github.com/rs/zerolog/log"
)

var (
	promptManager     *prompts.PromptManager
	promptManagerOnce sync.Once
	promptManagerErr  error
)

// GetPromptManager returns the shared prompt manager instance
func GetPromptManager() (*prompts.PromptManager, error) {
	promptManagerOnce.Do(func() {
		// Load config to get custom prompts directory
		cfg, err := config.Load()
		if err != nil {
			promptManagerErr = err
			return
		}

		// Create prompt manager with custom directory if specified
		promptManager, promptManagerErr = prompts.NewPromptManager(cfg.Prompts.CustomDir)
		if promptManagerErr != nil {
			log.Error().Err(promptManagerErr).Msg("Failed to initialize prompt manager")
		}
	})

	return promptManager, promptManagerErr
}
