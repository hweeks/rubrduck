package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeProject(t *testing.T) {
	tempDir := t.TempDir()

	// create sample files
	goFile := filepath.Join(tempDir, "main.go")
	os.WriteFile(goFile, []byte("package main"), 0644)

	pkgJSON := filepath.Join(tempDir, "package.json")
	os.WriteFile(pkgJSON, []byte(`{"dependencies":{"express":"^4"}}`), 0644)

	cfg := config.ProjectConfig{
		IgnorePaths:    []string{},
		CodeExtensions: []string{".go", ".js"},
	}

	info, err := AnalyzeProject(tempDir, cfg)
	require.NoError(t, err)

	assert.Contains(t, info.Languages, "Go")
	assert.Contains(t, info.Frameworks, "express")
	assert.Contains(t, info.Configs, "package.json")
	require.Greater(t, len(info.Files), 0)
}
