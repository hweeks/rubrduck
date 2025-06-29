package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGitRepo(t *testing.T) string {
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	require.NoError(t, err)

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	return tempDir
}

func TestGitTool_Status(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Test status on clean repository
	args := `{"operation": "status"}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "Working directory is clean")

	// Create a file and test status
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	args = `{"operation": "status"}`
	result, err = gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "Untracked")
	assert.Contains(t, result, "test.txt")
}

func TestGitTool_Diff(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("original content"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Modify the file
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Test diff
	args := `{"operation": "diff"}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "modified content")
	assert.Contains(t, result, "original content")
}

func TestGitTool_Commit(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Create a file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Test commit
	args := `{"operation": "commit", "args": "Test commit message"}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully committed changes")

	// Test commit with empty message
	args = `{"operation": "commit", "args": ""}`
	_, err = gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "commit message is required")
}

func TestGitTool_Branch(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Test listing branches
	args := `{"operation": "branch"}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	// Check for either 'main' or 'master' as default branch name
	assert.True(t, strings.Contains(result, "main") || strings.Contains(result, "master"), "Should contain main or master branch")

	// Test creating new branch
	args = `{"operation": "branch", "args": "create new-feature"}`
	result, err = gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "new-feature")

	// Test switching branch - use the actual default branch name
	defaultBranch := "main"
	if strings.Contains(result, "master") {
		defaultBranch = "master"
	}
	args = `{"operation": "branch", "args": "switch ` + defaultBranch + `"}`
	result, err = gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, defaultBranch)
}

func TestGitTool_Log(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Create and commit a file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Test log
	args := `{"operation": "log", "max_lines": 10}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "Initial commit")
}

func TestGitTool_Remote(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Test remote on repository without remotes
	args := `{"operation": "remote"}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "No remote repositories configured")
}

func TestGitTool_InvalidArguments(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Test invalid JSON
	_, err := gitTool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid arguments")

	// Test missing operation
	args := `{"args": "test"}`
	_, err = gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation is required")

	// Test unknown operation
	args = `{"operation": "unknown"}`
	_, err = gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown git operation")
}

func TestGitTool_GetDefinition(t *testing.T) {
	gitTool := NewGitTool("/tmp")
	def := gitTool.GetDefinition()

	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "git_operations", def.Function.Name)
	assert.Contains(t, def.Function.Description, "Git operations")

	// Check parameters
	params := def.Function.Parameters
	assert.NotNil(t, params["operation"])
	assert.NotNil(t, params["args"])
	assert.NotNil(t, params["file"])
	assert.NotNil(t, params["max_lines"])
}

func TestGitTool_FileSpecificOperations(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Create multiple files
	files := []string{"file1.txt", "file2.txt"}
	for _, file := range files {
		filePath := filepath.Join(tempDir, file)
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)
	}

	// Test status for specific file
	args := `{"operation": "status", "file": "file1.txt"}`
	result, err := gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "file1.txt")

	// Test diff for specific file
	cmd := exec.Command("git", "add", "file1.txt")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Add file1")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Modify file1
	file1Path := filepath.Join(tempDir, "file1.txt")
	err = os.WriteFile(file1Path, []byte("modified content"), 0644)
	require.NoError(t, err)

	args = `{"operation": "diff", "file": "file1.txt"}`
	result, err = gitTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "modified content")
}

func TestGitTool_BranchOperations(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Test invalid branch operations
	args := `{"operation": "branch", "args": "create"}`
	_, err := gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name required")

	args = `{"operation": "branch", "args": "switch"}`
	_, err = gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name required")

	args = `{"operation": "branch", "args": "delete"}`
	_, err = gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name required")

	args = `{"operation": "branch", "args": "unknown operation"}`
	_, err = gitTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown branch operation")
}

func TestGitTool_ContextCancellation(t *testing.T) {
	tempDir := setupGitRepo(t)
	gitTool := NewGitTool(tempDir)

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	args := `{"operation": "status"}`
	_, err := gitTool.Execute(ctx, args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}
