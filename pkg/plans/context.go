package plans

import (
	"fmt"
	"strings"
	"time"
)

// ContextFormatter formats plan context for AI consumption
type ContextFormatter struct {
	includeMetadata  bool
	includeRelated   bool
	maxContentLength int
}

// NewContextFormatter creates a new context formatter
func NewContextFormatter() *ContextFormatter {
	return &ContextFormatter{
		includeMetadata:  true,
		includeRelated:   true,
		maxContentLength: 2000, // Very conservative to avoid token overflow
	}
}

// SetIncludeMetadata controls whether metadata is included in context
func (cf *ContextFormatter) SetIncludeMetadata(include bool) {
	cf.includeMetadata = include
}

// SetIncludeRelated controls whether related plans are included
func (cf *ContextFormatter) SetIncludeRelated(include bool) {
	cf.includeRelated = include
}

// SetMaxContentLength sets the maximum content length to include
func (cf *ContextFormatter) SetMaxContentLength(length int) {
	cf.maxContentLength = length
}

// FormatContext formats plan context as a string for AI consumption
func (cf *ContextFormatter) FormatContext(context *PlanContext) string {
	if context == nil {
		return ""
	}

	var parts []string

	// Add current plan if available
	if context.CurrentPlan != nil {
		parts = append(parts, cf.formatCurrentPlan(context.CurrentPlan))
	}

	// Add related plans if requested
	if cf.includeRelated && len(context.RelatedPlans) > 0 {
		parts = append(parts, cf.formatRelatedPlans(context.RelatedPlans))
	}

	// Add mode context
	parts = append(parts, cf.formatModeContext(context.Mode))

	return strings.Join(parts, "\n\n")
}

// formatCurrentPlan formats the current plan for context
func (cf *ContextFormatter) formatCurrentPlan(plan *Plan) string {
	var parts []string

	// Add plan header
	header := fmt.Sprintf("## Current Plan: %s", plan.Metadata.Title)
	if plan.Metadata.Description != "" {
		header += fmt.Sprintf("\n**Description:** %s", plan.Metadata.Description)
	}
	parts = append(parts, header)

	// Add metadata if requested
	if cf.includeMetadata {
		metadata := cf.formatMetadata(&plan.Metadata)
		if metadata != "" {
			parts = append(parts, metadata)
		}
	}

	// Add content (truncated if necessary)
	content := plan.Content
	if len(content) > cf.maxContentLength {
		content = content[:cf.maxContentLength] + "\n\n[Content truncated...]"
	}
	parts = append(parts, fmt.Sprintf("### Plan Content:\n%s", content))

	return strings.Join(parts, "\n\n")
}

// formatRelatedPlans formats related plans for context
func (cf *ContextFormatter) formatRelatedPlans(plans []PlanSummary) string {
	if len(plans) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "## Related Plans:")

	for _, plan := range plans {
		planInfo := fmt.Sprintf("- **%s** (%s)", plan.Title, plan.Status)
		if plan.Description != "" {
			planInfo += fmt.Sprintf(": %s", plan.Description)
		}
		planInfo += fmt.Sprintf(" (Updated: %s)", plan.Updated.Format("2006-01-02"))
		parts = append(parts, planInfo)
	}

	return strings.Join(parts, "\n")
}

// formatMetadata formats plan metadata
func (cf *ContextFormatter) formatMetadata(metadata *PlanMetadata) string {
	var parts []string

	parts = append(parts, "### Plan Metadata:")
	parts = append(parts, fmt.Sprintf("- **ID:** %s", metadata.ID))
	parts = append(parts, fmt.Sprintf("- **Mode:** %s", metadata.Mode))
	parts = append(parts, fmt.Sprintf("- **Status:** %s", metadata.Status))
	parts = append(parts, fmt.Sprintf("- **Version:** %d", metadata.Version))
	parts = append(parts, fmt.Sprintf("- **Created:** %s", metadata.Created.Format(time.RFC3339)))
	parts = append(parts, fmt.Sprintf("- **Updated:** %s", metadata.Updated.Format(time.RFC3339)))

	if len(metadata.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("- **Tags:** %s", strings.Join(metadata.Tags, ", ")))
	}

	if metadata.Author != "" {
		parts = append(parts, fmt.Sprintf("- **Author:** %s", metadata.Author))
	}

	return strings.Join(parts, "\n")
}

// formatModeContext formats mode-specific context
func (cf *ContextFormatter) formatModeContext(mode string) string {
	modeDescriptions := map[string]string{
		"planning":  "You are in **Planning Mode**. Focus on creating comprehensive project plans, architectural decisions, and implementation roadmaps.",
		"building":  "You are in **Building Mode**. Focus on implementing features, generating code, and following existing plans.",
		"debugging": "You are in **Debugging Mode**. Focus on analyzing errors, tracing issues, and providing systematic solutions.",
		"enhance":   "You are in **Enhance Mode**. Focus on improving code quality, refactoring, and modernization.",
	}

	if description, exists := modeDescriptions[mode]; exists {
		return fmt.Sprintf("## Mode Context:\n%s", description)
	}

	return fmt.Sprintf("## Mode Context:\nYou are in **%s** mode.", mode)
}

// FormatContextForPrompt formats context specifically for prompt injection
func (cf *ContextFormatter) FormatContextForPrompt(context *PlanContext) string {
	if context == nil {
		return ""
	}

	formatter := NewContextFormatter()
	formatter.SetIncludeMetadata(false) // Don't include metadata in prompts
	formatter.SetMaxContentLength(500)  // Shorter for prompts

	return formatter.FormatContext(context)
}

// FormatContextForSummary formats context for summary/overview purposes
func (cf *ContextFormatter) FormatContextForSummary(context *PlanContext) string {
	if context == nil {
		return ""
	}

	var parts []string

	// Add current plan summary
	if context.CurrentPlan != nil {
		parts = append(parts, fmt.Sprintf("**Current Plan:** %s (%s)",
			context.CurrentPlan.Metadata.Title,
			context.CurrentPlan.Metadata.Status))
	}

	// Add related plans count
	if len(context.RelatedPlans) > 0 {
		parts = append(parts, fmt.Sprintf("**Related Plans:** %d plans available", len(context.RelatedPlans)))
	}

	// Add mode
	parts = append(parts, fmt.Sprintf("**Mode:** %s", context.Mode))

	return strings.Join(parts, " | ")
}

// GetContextSnippet returns a brief snippet of plan content for quick reference
func (cf *ContextFormatter) GetContextSnippet(context *PlanContext, maxLength int) string {
	if context == nil || context.CurrentPlan == nil {
		return ""
	}

	content := context.CurrentPlan.Content
	if len(content) <= maxLength {
		return content
	}

	// Try to find a good breaking point
	truncated := content[:maxLength]
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > maxLength/2 {
		return truncated[:lastNewline] + "\n[Content truncated...]"
	}

	return truncated + "\n[Content truncated...]"
}
