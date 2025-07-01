package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/hammie/rubrduck/internal/project"
	"github.com/rs/zerolog/log"
)

// ProjectTool analyzes the current project structure
// and returns detected languages, frameworks, and files.
type ProjectTool struct {
	basePath string
	cfg      config.ProjectConfig
}

// NewProjectTool creates a new project analysis tool
func NewProjectTool(basePath string, cfg config.ProjectConfig) *ProjectTool {
	return &ProjectTool{basePath: basePath, cfg: cfg}
}

func (p *ProjectTool) GetDefinition() ai.Tool {
	return ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "project_analyze",
			Description: "Analyze the project structure and configuration",
			Parameters:  map[string]interface{}{},
		},
	}
}

func (p *ProjectTool) Execute(ctx context.Context, args string) (string, error) {
	info, err := project.AnalyzeProject(p.basePath, p.cfg)
	if err != nil {
		return "", fmt.Errorf("analysis failed: %w", err)
	}
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	log.Debug().Msg("project analysis completed")
	return string(data), nil
}
