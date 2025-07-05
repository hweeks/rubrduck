package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed defaults/*.yaml
var defaultTemplates embed.FS

// PromptTemplate represents a single prompt template
type PromptTemplate struct {
	Name            string            `yaml:"name"`
	Description     string            `yaml:"description"`
	SystemPrompt    string            `yaml:"system_prompt"`
	ToolingPreamble string            `yaml:"tooling_preamble,omitempty"`
	Variables       map[string]string `yaml:"variables,omitempty"`
}

// PromptManager manages loading and accessing prompt templates
type PromptManager struct {
	templates       map[string]*PromptTemplate
	customDir       string
	toolingPreamble string
}

// NewPromptManager creates a new prompt manager
func NewPromptManager(customDir string) (*PromptManager, error) {
	pm := &PromptManager{
		templates: make(map[string]*PromptTemplate),
		customDir: customDir,
	}

	// Load default templates
	if err := pm.loadDefaultTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load default templates: %w", err)
	}

	// Load custom templates if directory is specified
	if customDir != "" {
		if err := pm.loadCustomTemplates(customDir); err != nil {
			return nil, fmt.Errorf("failed to load custom templates: %w", err)
		}
	}

	// Load tooling preamble
	if err := pm.loadToolingPreamble(); err != nil {
		return nil, fmt.Errorf("failed to load tooling preamble: %w", err)
	}

	return pm, nil
}

// loadDefaultTemplates loads templates from embedded files
func (pm *PromptManager) loadDefaultTemplates() error {
	entries, err := defaultTemplates.ReadDir("defaults")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Skip tooling preamble file
		if entry.Name() == "tooling_preamble.yaml" {
			continue
		}

		data, err := defaultTemplates.ReadFile(filepath.Join("defaults", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", entry.Name(), err)
		}

		var tmpl PromptTemplate
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			return fmt.Errorf("failed to parse template %s: %w", entry.Name(), err)
		}

		// Store by name without extension
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		pm.templates[name] = &tmpl
	}

	return nil
}

// loadCustomTemplates loads templates from a custom directory
func (pm *PromptManager) loadCustomTemplates(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Custom directory doesn't exist, that's okay
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Skip tooling preamble file
		if entry.Name() == "tooling_preamble.yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read custom template %s: %w", entry.Name(), err)
		}

		var tmpl PromptTemplate
		if err := yaml.Unmarshal(data, &tmpl); err != nil {
			return fmt.Errorf("failed to parse custom template %s: %w", entry.Name(), err)
		}

		// Override default template if exists
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		pm.templates[name] = &tmpl
	}

	return nil
}

// loadToolingPreamble loads the tooling preamble template
func (pm *PromptManager) loadToolingPreamble() error {
	// Try to load from custom directory first
	if pm.customDir != "" {
		customPath := filepath.Join(pm.customDir, "tooling_preamble.yaml")
		if data, err := os.ReadFile(customPath); err == nil {
			var preamble struct {
				ToolingPreamble string `yaml:"tooling_preamble"`
			}
			if err := yaml.Unmarshal(data, &preamble); err == nil {
				pm.toolingPreamble = preamble.ToolingPreamble
				return nil
			}
		}
	}

	// Load default tooling preamble
	data, err := defaultTemplates.ReadFile("defaults/tooling_preamble.yaml")
	if err != nil {
		return fmt.Errorf("failed to read default tooling preamble: %w", err)
	}

	var preamble struct {
		ToolingPreamble string `yaml:"tooling_preamble"`
	}
	if err := yaml.Unmarshal(data, &preamble); err != nil {
		return fmt.Errorf("failed to parse tooling preamble: %w", err)
	}

	pm.toolingPreamble = preamble.ToolingPreamble
	return nil
}

// GetPrompt returns a formatted prompt for the given mode
func (pm *PromptManager) GetPrompt(mode string, variables map[string]string) (string, error) {
	tmpl, ok := pm.templates[mode]
	if !ok {
		return "", fmt.Errorf("template not found for mode: %s", mode)
	}

	// Merge template variables with provided variables
	vars := make(map[string]string)
	for k, v := range tmpl.Variables {
		vars[k] = v
	}
	for k, v := range variables {
		vars[k] = v
	}

	// Parse and execute the template
	t, err := template.New(mode).Parse(tmpl.SystemPrompt)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Combine system prompt with tooling preamble
	result := buf.String()
	if tmpl.ToolingPreamble != "" {
		result = result + "\n\n" + tmpl.ToolingPreamble
	} else if pm.toolingPreamble != "" {
		result = result + "\n\n" + pm.toolingPreamble
	}

	return result, nil
}

// ListModes returns all available prompt modes
func (pm *PromptManager) ListModes() []string {
	modes := make([]string, 0, len(pm.templates))
	for mode := range pm.templates {
		modes = append(modes, mode)
	}
	return modes
}

// GetTemplate returns the raw template for a mode
func (pm *PromptManager) GetTemplate(mode string) (*PromptTemplate, bool) {
	tmpl, ok := pm.templates[mode]
	return tmpl, ok
}
