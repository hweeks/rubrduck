package tui2

import (
	"context"
	"fmt"
)

// GetDebuggingSystemPrompt returns the system prompt for debugging mode
func GetDebuggingSystemPrompt() (string, error) {
	pm, err := GetPromptManager()
	if err != nil {
		return "", fmt.Errorf("failed to get prompt manager: %w", err)
	}

	prompt, err := pm.GetPrompt("debugging", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get debugging prompt: %w", err)
	}

	return prompt, nil
}

// ProcessDebuggingRequest handles AI requests for debugging mode using the agent
func ProcessDebuggingRequest(ctx context.Context, agent AgentInterface, userInput, model string) (string, error) {
	// Clear agent history and set system context
	agent.ClearHistory()

	// Get system prompt
	systemPrompt, err := GetDebuggingSystemPrompt()
	if err != nil {
		return "", fmt.Errorf("failed to get debugging system prompt: %w", err)
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
