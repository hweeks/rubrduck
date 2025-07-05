package tui2

import (
	"context"
	"fmt"

	"github.com/hammie/rubrduck/internal/agent"
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
func ProcessDebuggingRequest(ctx context.Context, agent *agent.Agent, userInput, model string) (<-chan agent.StreamEvent, error) {
	systemPrompt, err := GetDebuggingSystemPrompt()
	if err != nil {
		return nil, err
	}

	contextualInput := fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)
	return agent.StreamEvents(ctx, contextualInput)
}
