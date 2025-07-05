package tui2

import (
	"context"
	"fmt"

	"github.com/hammie/rubrduck/internal/agent"
	"github.com/hammie/rubrduck/pkg/plans"
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

	// Get planning context for building mode
	planContext, err := getBuildingContext()
	if err != nil {
		// Log error but continue without context
		fmt.Printf("Warning: failed to get building context: %v\n", err)
	}

	// Build contextual input
	var contextualInput string
	if planContext != nil && (planContext.CurrentPlan != nil || len(planContext.RelatedPlans) > 0) {
		formatter := plans.NewContextFormatter()
		formatter.SetIncludeMetadata(false)
		formatter.SetMaxContentLength(1000)
		contextStr := formatter.FormatContext(planContext)

		contextualInput = fmt.Sprintf("System context: %s\n\nPlan Context:\n%s\n\nUser request: %s",
			systemPrompt, contextStr, userInput)
	} else {
		contextualInput = fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)
	}

	return agent.StreamEvents(ctx, contextualInput)
}

// getBuildingContext retrieves relevant plan context for building mode
func getBuildingContext() (*plans.PlanContext, error) {
	pm, err := GetPlanManager()
	if err != nil {
		return nil, err
	}

	// Get context for building mode, including planning plans
	return pm.GetContext("building", "")
}
