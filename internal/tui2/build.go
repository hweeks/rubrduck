package tui2

import (
	"context"
	"fmt"

	"github.com/hammie/rubrduck/internal/agent"
)

// GetBuildingSystemPrompt returns the system prompt for building mode
func GetBuildingSystemPrompt() (string, error) {
	pm, err := GetPromptManager()
	if err != nil {
		return "", fmt.Errorf("failed to get prompt manager: %w", err)
	}

	prompt, err := pm.GetPrompt("building", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get building prompt: %w", err)
	}

	return prompt, nil
}

// ProcessBuildingRequest handles AI requests for building mode using the agent
func ProcessBuildingRequest(ctx context.Context, agent *agent.Agent, userInput, model string) (<-chan agent.StreamEvent, error) {
	systemPrompt, err := GetBuildingSystemPrompt()
	if err != nil {
		return nil, err
	}

	contextualInput := fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)
	return agent.StreamEvents(ctx, contextualInput)
}
