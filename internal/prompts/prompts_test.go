package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPromptManager(t *testing.T) {
	tests := []struct {
		name      string
		customDir string
		wantErr   bool
		wantModes []string
	}{
		{
			name:      "load default templates",
			customDir: "",
			wantErr:   false,
			wantModes: []string{"planning", "building", "debugging", "enhance"},
		},
		{
			name:      "load with non-existent custom dir",
			customDir: "/non/existent/path",
			wantErr:   false,
			wantModes: []string{"planning", "building", "debugging", "enhance"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm, err := NewPromptManager(tt.customDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPromptManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if pm == nil {
				t.Fatal("expected PromptManager to be non-nil")
			}

			modes := pm.ListModes()
			if len(modes) != len(tt.wantModes) {
				t.Errorf("expected %d modes, got %d", len(tt.wantModes), len(modes))
			}

			// Check that all expected modes are present
			modeMap := make(map[string]bool)
			for _, mode := range modes {
				modeMap[mode] = true
			}

			for _, wantMode := range tt.wantModes {
				if !modeMap[wantMode] {
					t.Errorf("expected mode %s not found", wantMode)
				}
			}
		})
	}
}

func TestPromptManagerGetPrompt(t *testing.T) {
	pm, err := NewPromptManager("")
	if err != nil {
		t.Fatalf("failed to create prompt manager: %v", err)
	}

	tests := []struct {
		name      string
		mode      string
		variables map[string]string
		wantErr   bool
		contains  []string
	}{
		{
			name:      "get planning prompt",
			mode:      "planning",
			variables: nil,
			wantErr:   false,
			contains:  []string{"RubrDuck", "PLANNING MODE", "TOOLS AVAILABLE"},
		},
		{
			name:      "get building prompt",
			mode:      "building",
			variables: nil,
			wantErr:   false,
			contains:  []string{"RubrDuck", "BUILDING MODE", "TOOLS AVAILABLE"},
		},
		{
			name:      "get debugging prompt",
			mode:      "debugging",
			variables: nil,
			wantErr:   false,
			contains:  []string{"RubrDuck", "DEBUGGING MODE", "TOOLS AVAILABLE"},
		},
		{
			name:      "get enhance prompt",
			mode:      "enhance",
			variables: nil,
			wantErr:   false,
			contains:  []string{"RubrDuck", "ENHANCE MODE", "TOOLS AVAILABLE"},
		},
		{
			name:      "get non-existent prompt",
			mode:      "nonexistent",
			variables: nil,
			wantErr:   true,
			contains:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := pm.GetPrompt(tt.mode, tt.variables)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPrompt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				for _, expected := range tt.contains {
					if !strings.Contains(prompt, expected) {
						t.Errorf("expected prompt to contain %q, but it didn't", expected)
					}
				}
			}
		})
	}
}

func TestPromptManagerCustomTemplates(t *testing.T) {
	// Create a temporary directory for custom templates
	tempDir, err := os.MkdirTemp("", "prompt_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a custom planning template
	customTemplate := `name: "Custom Planning"
description: "Custom planning template"
system_prompt: |
  This is a custom planning prompt.
  It overrides the default planning prompt.`

	err = os.WriteFile(filepath.Join(tempDir, "planning.yaml"), []byte(customTemplate), 0644)
	if err != nil {
		t.Fatalf("failed to write custom template: %v", err)
	}

	// Create a new mode template
	newModeTemplate := `name: "Testing Mode"
description: "A new testing mode"
system_prompt: |
  This is a testing mode prompt.
  It adds a new mode to the system.`

	err = os.WriteFile(filepath.Join(tempDir, "testing.yaml"), []byte(newModeTemplate), 0644)
	if err != nil {
		t.Fatalf("failed to write new mode template: %v", err)
	}

	// Create prompt manager with custom directory
	pm, err := NewPromptManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create prompt manager with custom dir: %v", err)
	}

	// Test that custom planning template overrides default
	prompt, err := pm.GetPrompt("planning", nil)
	if err != nil {
		t.Errorf("failed to get custom planning prompt: %v", err)
	}

	if !strings.Contains(prompt, "This is a custom planning prompt") {
		t.Error("expected custom planning prompt, got default")
	}

	// Test that new mode is available
	if _, ok := pm.GetTemplate("testing"); !ok {
		t.Error("expected testing mode to be available")
	}

	// Test that other modes still have defaults
	buildPrompt, err := pm.GetPrompt("building", nil)
	if err != nil {
		t.Errorf("failed to get building prompt: %v", err)
	}

	if !strings.Contains(buildPrompt, "BUILDING MODE") {
		t.Error("expected default building prompt")
	}
}

func TestPromptManagerTemplateVariables(t *testing.T) {
	// Create a temporary directory for custom templates
	tempDir, err := os.MkdirTemp("", "prompt_var_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a template with variables
	varTemplate := `name: "Variable Test"
description: "Template with variables"
system_prompt: |
  Hello {{.Name}}!
  Your task is: {{.Task}}
  Default value: {{.Default}}
variables:
  Default: "default value"`

	err = os.WriteFile(filepath.Join(tempDir, "vartest.yaml"), []byte(varTemplate), 0644)
	if err != nil {
		t.Fatalf("failed to write variable template: %v", err)
	}

	pm, err := NewPromptManager(tempDir)
	if err != nil {
		t.Fatalf("failed to create prompt manager: %v", err)
	}

	// Test with provided variables
	variables := map[string]string{
		"Name": "Alice",
		"Task": "Write tests",
	}

	prompt, err := pm.GetPrompt("vartest", variables)
	if err != nil {
		t.Errorf("failed to get prompt with variables: %v", err)
	}

	expectedStrings := []string{
		"Hello Alice!",
		"Your task is: Write tests",
		"Default value: default value",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(prompt, expected) {
			t.Errorf("expected prompt to contain %q, but it didn't", expected)
		}
	}
}

func TestPromptManagerToolingPreamble(t *testing.T) {
	pm, err := NewPromptManager("")
	if err != nil {
		t.Fatalf("failed to create prompt manager: %v", err)
	}

	// Get a prompt and check it includes tooling preamble
	prompt, err := pm.GetPrompt("planning", nil)
	if err != nil {
		t.Fatalf("failed to get planning prompt: %v", err)
	}

	// Check for tooling preamble content
	toolingKeywords := []string{
		"FILE OPERATIONS",
		"SHELL EXECUTION",
		"GIT OPERATIONS",
		"file_operations",
		"shell_execute",
		"git_operations",
	}

	for _, keyword := range toolingKeywords {
		if !strings.Contains(prompt, keyword) {
			t.Errorf("expected tooling preamble to contain %q", keyword)
		}
	}
}

func TestGetTemplate(t *testing.T) {
	pm, err := NewPromptManager("")
	if err != nil {
		t.Fatalf("failed to create prompt manager: %v", err)
	}

	// Test existing template
	tmpl, ok := pm.GetTemplate("planning")
	if !ok {
		t.Error("expected planning template to exist")
	}
	if tmpl == nil {
		t.Error("expected template to be non-nil")
		return
	}
	if tmpl.Name != "Planning Mode" {
		t.Errorf("expected template name to be 'Planning Mode', got %s", tmpl.Name)
	}

	// Test non-existing template
	_, ok = pm.GetTemplate("nonexistent")
	if ok {
		t.Error("expected nonexistent template to not exist")
	}
}

func TestListModes(t *testing.T) {
	pm, err := NewPromptManager("")
	if err != nil {
		t.Fatalf("failed to create prompt manager: %v", err)
	}

	modes := pm.ListModes()
	if len(modes) < 4 {
		t.Errorf("expected at least 4 modes, got %d", len(modes))
	}

	// Check that standard modes are present
	expectedModes := map[string]bool{
		"planning":  false,
		"building":  false,
		"debugging": false,
		"enhance":   false,
	}

	for _, mode := range modes {
		if _, ok := expectedModes[mode]; ok {
			expectedModes[mode] = true
		}
	}

	for mode, found := range expectedModes {
		if !found {
			t.Errorf("expected mode %s not found in list", mode)
		}
	}
}
