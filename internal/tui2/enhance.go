package tui2

import (
	"context"
	"fmt"
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
func ProcessEnhanceRequest(ctx context.Context, agent AgentInterface, userInput, model string) (string, error) {
	// Clear agent history and set system context
	agent.ClearHistory()

	// Get system prompt
	systemPrompt, err := GetEnhanceSystemPrompt()
	if err != nil {
		return "", fmt.Errorf("failed to get enhance system prompt: %w", err)
	}

	// Create a combined input with system context
	contextualInput := fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)

	// Use agent.Chat which has access to tools including file reading
	response, err := agent.Chat(ctx, contextualInput)
	if err != nil {
		return "", err
	}

	return response, nil
}
