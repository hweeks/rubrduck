package tui2

import (
	"context"
	"fmt"

	"github.com/hammie/rubrduck/internal/agent"
)

// GetPlanningSystemPrompt returns the system prompt for planning mode
func GetPlanningSystemPrompt() (string, error) {
	pm, err := GetPromptManager()
	if err != nil {
		return "", fmt.Errorf("failed to get prompt manager: %w", err)
	}

	prompt, err := pm.GetPrompt("planning", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get planning prompt: %w", err)
	}

	return prompt, nil
}

// ProcessPlanningRequest handles AI requests for planning mode using the agent
func ProcessPlanningRequest(ctx context.Context, agent *agent.Agent, userInput, model string) (<-chan agent.StreamEvent, error) {
	systemPrompt, err := GetPlanningSystemPrompt()
	if err != nil {
		return nil, err
	}

	contextualInput := fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)
	return agent.StreamEvents(ctx, contextualInput)
}
