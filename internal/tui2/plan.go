package tui2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hammie/rubrduck/internal/agent"
	"github.com/hammie/rubrduck/pkg/plans"
)

var planManager *plans.Manager

// GetPlanManager returns the global plan manager instance
func GetPlanManager() (*plans.Manager, error) {
	if planManager == nil {
		// Initialize plan manager with .duckie directory in current working directory
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}

		duckieDir := filepath.Join(wd, ".duckie")
		planManager = plans.NewManager(duckieDir)

		if err := planManager.Initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize plan manager: %w", err)
		}
	}
	return planManager, nil
}

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

	// Get plan context
	planContext, err := getPlanningContext()
	if err != nil {
		// Log error but continue without context
		fmt.Printf("Warning: failed to get plan context: %v\n", err)
	}

	// Build contextual input
	var contextualInput string
	if planContext != nil && len(planContext.RelatedPlans) > 0 {
		formatter := plans.NewContextFormatter()
		formatter.SetIncludeMetadata(false)
		formatter.SetMaxContentLength(500)
		contextStr := formatter.FormatContext(planContext)

		contextualInput = fmt.Sprintf("System context: %s\n\nPlan Context:\n%s\n\nUser request: %s",
			systemPrompt, contextStr, userInput)
	} else {
		contextualInput = fmt.Sprintf("System context: %s\n\nUser request: %s", systemPrompt, userInput)
	}

	return agent.StreamEvents(ctx, contextualInput)
}

// getPlanningContext retrieves relevant plan context for planning mode
func getPlanningContext() (*plans.PlanContext, error) {
	pm, err := GetPlanManager()
	if err != nil {
		return nil, err
	}

	// Get context for planning mode, including related plans
	return pm.GetContext("planning", "")
}

// SavePlanningResponse saves the AI response as a new plan
func SavePlanningResponse(title, description, content string) (*plans.Plan, error) {
	pm, err := GetPlanManager()
	if err != nil {
		return nil, err
	}

	// Create a new plan from the AI response
	plan, err := pm.CreatePlan("planning", title, description, content)
	if err != nil {
		return nil, fmt.Errorf("failed to save planning response: %w", err)
	}

	return plan, nil
}

// GetLatestPlanningPlan returns the most recent planning plan
func GetLatestPlanningPlan() (*plans.Plan, error) {
	pm, err := GetPlanManager()
	if err != nil {
		return nil, err
	}

	return pm.GetLatestPlan("planning")
}

// ListPlanningPlans returns all planning plans
func ListPlanningPlans() ([]plans.PlanSummary, error) {
	pm, err := GetPlanManager()
	if err != nil {
		return nil, err
	}

	filter := &plans.PlanFilter{Mode: "planning"}
	return pm.ListPlans(filter)
}

// SearchPlanningPlans searches for planning plans
func SearchPlanningPlans(query string) ([]plans.PlanSummary, error) {
	pm, err := GetPlanManager()
	if err != nil {
		return nil, err
	}

	filter := &plans.PlanFilter{Mode: "planning"}
	return pm.SearchPlans(query, filter)
}
