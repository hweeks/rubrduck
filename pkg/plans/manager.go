package plans

import (
	"fmt"
	"strings"
	"time"
)

// Manager implements the PlanManager interface
type Manager struct {
	storage *FileStorage
}

// NewManager creates a new plan manager
func NewManager(baseDir string) *Manager {
	return &Manager{
		storage: NewFileStorage(baseDir),
	}
}

// Initialize sets up the plan management system
func (m *Manager) Initialize() error {
	return m.storage.Initialize()
}

// CreatePlan creates a new plan
func (m *Manager) CreatePlan(mode, title, description string, content string) (*Plan, error) {
	// Validate mode
	if !isValidMode(mode) {
		return nil, fmt.Errorf("invalid mode: %s", mode)
	}

	// Create plan
	plan := &Plan{
		Metadata: PlanMetadata{
			Title:       title,
			Mode:        mode,
			Description: description,
			Status:      StatusDraft,
			Version:     1,
		},
		Content: content,
	}

	// Save plan
	if err := m.storage.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save plan: %w", err)
	}

	return plan, nil
}

// GetPlan retrieves a plan by ID
func (m *Manager) GetPlan(id string) (*Plan, error) {
	return m.storage.LoadPlan(id)
}

// UpdatePlan updates an existing plan
func (m *Manager) UpdatePlan(id string, content string) (*Plan, error) {
	// Load existing plan
	plan, err := m.storage.LoadPlan(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan: %w", err)
	}

	// Update content and version
	plan.Content = content
	plan.Metadata.Version++
	plan.Metadata.Updated = time.Now()

	// Save updated plan
	if err := m.storage.SavePlan(plan); err != nil {
		return nil, fmt.Errorf("failed to save updated plan: %w", err)
	}

	return plan, nil
}

// DeletePlan removes a plan
func (m *Manager) DeletePlan(id string) error {
	return m.storage.DeletePlan(id)
}

// ListPlans returns plan summaries with optional filtering
func (m *Manager) ListPlans(filter *PlanFilter) ([]PlanSummary, error) {
	summaries, err := m.storage.ListPlans()
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return summaries, nil
	}

	// Apply filters
	var filtered []PlanSummary
	for _, summary := range summaries {
		if m.matchesFilter(summary, filter) {
			filtered = append(filtered, summary)
		}
	}

	return filtered, nil
}

// SearchPlans performs a text search across plan titles and content
func (m *Manager) SearchPlans(query string, filter *PlanFilter) ([]PlanSummary, error) {
	summaries, err := m.ListPlans(filter)
	if err != nil {
		return nil, err
	}

	if query == "" {
		return summaries, nil
	}

	query = strings.ToLower(query)
	var results []PlanSummary

	for _, summary := range summaries {
		// Search in title and description
		if strings.Contains(strings.ToLower(summary.Title), query) ||
			strings.Contains(strings.ToLower(summary.Description), query) {
			results = append(results, summary)
			continue
		}

		// Search in content
		plan, err := m.GetPlan(summary.ID)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(plan.Content), query) {
			results = append(results, summary)
		}
	}

	return results, nil
}

// GetContext provides context for AI calls
func (m *Manager) GetContext(mode string, planID string) (*PlanContext, error) {
	context := &PlanContext{
		Mode: mode,
	}

	// Get current plan if specified
	if planID != "" {
		plan, err := m.GetPlan(planID)
		if err != nil {
			return nil, fmt.Errorf("failed to load current plan: %w", err)
		}
		context.CurrentPlan = plan
	}

	// Get related plans for the mode
	filter := &PlanFilter{Mode: mode}
	relatedPlans, err := m.ListPlans(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to load related plans: %w", err)
	}
	context.RelatedPlans = relatedPlans

	return context, nil
}

// GetLatestPlan returns the most recent plan for a mode
func (m *Manager) GetLatestPlan(mode string) (*Plan, error) {
	return m.storage.GetLatestPlanByMode(mode)
}

// GetTemplates returns available templates for a mode
func (m *Manager) GetTemplates(mode string) ([]PlanTemplate, error) {
	// For now, return built-in templates
	// In the future, this could load from template files
	return getBuiltInTemplates(mode), nil
}

// CreateFromTemplate creates a plan from a template
func (m *Manager) CreateFromTemplate(templateName, title, description string) (*Plan, error) {
	// Find template
	templates, err := m.GetTemplates("")
	if err != nil {
		return nil, err
	}

	var template *PlanTemplate
	for _, t := range templates {
		if t.Name == templateName {
			template = &t
			break
		}
	}

	if template == nil {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// Create plan from template
	return m.CreatePlan(template.Mode, title, description, template.Content)
}

// ValidatePlan validates a plan structure
func (m *Manager) ValidatePlan(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("plan cannot be nil")
	}

	if plan.Metadata.Title == "" {
		return fmt.Errorf("plan title is required")
	}

	if !isValidMode(plan.Metadata.Mode) {
		return fmt.Errorf("invalid mode: %s", plan.Metadata.Mode)
	}

	if plan.Content == "" {
		return fmt.Errorf("plan content is required")
	}

	return nil
}

// matchesFilter checks if a plan summary matches the given filter
func (m *Manager) matchesFilter(summary PlanSummary, filter *PlanFilter) bool {
	if filter.Mode != "" && summary.Mode != filter.Mode {
		return false
	}

	if filter.Status != "" && summary.Status != filter.Status {
		return false
	}

	if !filter.From.IsZero() && summary.Created.Before(filter.From) {
		return false
	}

	if !filter.To.IsZero() && summary.Created.After(filter.To) {
		return false
	}

	// TODO: Implement tag filtering
	// if len(filter.Tags) > 0 {
	//     // Check if any of the filter tags match plan tags
	// }

	return true
}

// isValidMode checks if a mode is supported
func isValidMode(mode string) bool {
	for _, supportedMode := range SupportedModes {
		if supportedMode == mode {
			return true
		}
	}
	return false
}

// getBuiltInTemplates returns built-in templates for a mode
func getBuiltInTemplates(mode string) []PlanTemplate {
	templates := []PlanTemplate{
		{
			Name:        "standard-planning",
			Mode:        "planning",
			Description: "Standard planning template with executive summary and phases",
			Content: `# Executive Summary

Brief overview of the project goals and expected outcomes.

## Technical Analysis

Summary of architectural decisions, technology choices, and key considerations.

## Implementation Roadmap

### Phase 1: [Phase Name]
- [ ] Task 1
- [ ] Task 2

### Phase 2: [Phase Name]
- [ ] Task 1
- [ ] Task 2

## Risk Assessment

Identify critical risks and mitigation strategies.

## Validation Strategy

Methods to test and validate each phase.`,
		},
		{
			Name:        "standard-building",
			Mode:        "building",
			Description: "Standard building template for implementation tasks",
			Content: `# Implementation Plan

## Overview
Brief description of what will be implemented.

## Requirements
- Requirement 1
- Requirement 2

## Implementation Steps
1. Step 1
2. Step 2

## Testing Strategy
How the implementation will be tested.

## Success Criteria
How we'll know the implementation is complete.`,
		},
	}

	// Filter by mode if specified
	if mode != "" {
		var filtered []PlanTemplate
		for _, template := range templates {
			if template.Mode == mode {
				filtered = append(filtered, template)
			}
		}
		return filtered
	}

	return templates
}
