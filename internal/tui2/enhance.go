package tui2

import (
	"context"
	"fmt"

	"github.com/hammie/rubrduck/internal/agent"
)

// GetEnhanceSystemPrompt returns the system prompt for enhancement mode
func GetEnhanceSystemPrompt() (string, error) {
	pm, err := GetPromptManager()
	if err != nil {
		return "", fmt.Errorf("failed to get prompt manager: %w", err)
	}

	prompt, err := pm.GetPrompt("enhance", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get enhance prompt: %w", err)
	}

	return prompt, nil
}

// ProcessEnhanceRequest handles AI requests for enhancement mode using the agent
func ProcessEnhanceRequest(ctx context.Context, agent *agent.Agent, userInput, model string) (<-chan agent.StreamEvent, error) {
	systemPrompt, err := GetEnhanceSystemPrompt()
	if err != nil {
		return nil, err
	}

	contextualInput := fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)
	return agent.StreamEvents(ctx, contextualInput)
}
