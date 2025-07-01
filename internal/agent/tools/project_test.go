package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectTool_Execute(t *testing.T) {
	tempDir := t.TempDir()
	// sample go file
	goFile := filepath.Join(tempDir, "main.go")
	require.NoError(t, os.WriteFile(goFile, []byte("package main"), 0644))

	cfg := config.ProjectConfig{CodeExtensions: []string{".go"}}
	tool := NewProjectTool(tempDir, cfg)

	result, err := tool.Execute(context.Background(), "{}")
	require.NoError(t, err)

	var info map[string]interface{}
	err = json.Unmarshal([]byte(result), &info)
	require.NoError(t, err)

	_, ok := info["languages"]
	assert.True(t, ok)
}

func TestProjectTool_GetDefinition(t *testing.T) {
	tool := NewProjectTool("/tmp", config.ProjectConfig{})
	def := tool.GetDefinition()
	assert.Equal(t, "project_analyze", def.Function.Name)
}
