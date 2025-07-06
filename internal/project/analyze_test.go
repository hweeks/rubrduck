package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyze(t *testing.T) {
	dir := t.TempDir()

	// create sample files and config
	goFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(goFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("write go file: %v", err)
	}
	modFile := filepath.Join(dir, "go.mod")
	modContent := "module example.com/test\n\nrequire github.com/gin-gonic/gin v1.0.0"
	if err := os.WriteFile(modFile, []byte(modContent), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	pkgJSON := filepath.Join(dir, "package.json")
	pkgContent := `{"dependencies":{"react":"^18.0.0"}}`
	if err := os.WriteFile(pkgJSON, []byte(pkgContent), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	analysis, err := Analyze(dir)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}

	if analysis.Languages["Go"] != 1 {
		t.Errorf("expected 1 Go file, got %d", analysis.Languages["Go"])
	}

	if len(analysis.Frameworks) == 0 {
		t.Errorf("expected at least one framework detected")
	}
}
