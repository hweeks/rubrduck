package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileTool_ReadFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Create a test file
	testContent := "Hello, World!\nThis is a test file."
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	// Test reading file
	args := `{"type": "read", "path": "test.txt"}`
	result, err := fileTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, testContent)
}

func TestFileTool_WriteFile(t *testing.T) {
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Test writing new file
	content := "New file content"
	args := `{"type": "write", "path": "newfile.txt", "content": "New file content"}`
	result, err := fileTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "Successfully wrote")

	// Verify file was created
	filePath := filepath.Join(tempDir, "newfile.txt")
	readContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(readContent))
}

func TestFileTool_ListDirectory(t *testing.T) {
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Create some test files
	files := []string{"file1.txt", "file2.txt", "dir1"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if file == "dir1" {
			err := os.Mkdir(path, 0755)
			require.NoError(t, err)
		} else {
			err := os.WriteFile(path, []byte("test"), 0644)
			require.NoError(t, err)
		}
	}

	// Test listing directory
	args := `{"type": "list", "path": ".", "max_results": 10}`
	result, err := fileTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "file1.txt")
	assert.Contains(t, result, "file2.txt")
	assert.Contains(t, result, "dir1")
}

func TestFileTool_SearchFiles(t *testing.T) {
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Create test files
	files := []string{"test1.txt", "test2.txt", "other.txt", "subdir/test3.txt"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		dir := filepath.Dir(path)
		if dir != tempDir {
			err := os.MkdirAll(dir, 0755)
			require.NoError(t, err)
		}
		err := os.WriteFile(path, []byte("test"), 0644)
		require.NoError(t, err)
	}

	// Test searching files
	args := `{"type": "search", "path": ".", "pattern": "test", "max_results": 10}`
	result, err := fileTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "test1.txt")
	assert.Contains(t, result, "test2.txt")
	assert.Contains(t, result, "test3.txt")
	assert.NotContains(t, result, "other.txt")
}

func TestFileTool_PathSanitization(t *testing.T) {
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Test path traversal attempt
	args := `{"type": "read", "path": "../../../etc/passwd"}`
	_, err := fileTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path outside project bounds")

	// Test absolute path
	args = `{"type": "read", "path": "/etc/passwd"}`
	_, err = fileTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path outside project bounds")
}

func TestFileTool_LargeFileHandling(t *testing.T) {
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Create a large file (> 1MB)
	largeContent := make([]byte, 2*1024*1024) // 2MB
	for i := range largeContent {
		largeContent[i] = 'A'
	}

	testFile := filepath.Join(tempDir, "large.txt")
	err := os.WriteFile(testFile, largeContent, 0644)
	require.NoError(t, err)

	// Test reading large file
	args := `{"type": "read", "path": "large.txt"}`
	result, err := fileTool.Execute(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "File too large")
	assert.Contains(t, result, "Showing first 1KB")
}

func TestFileTool_InvalidArguments(t *testing.T) {
	tempDir := t.TempDir()
	fileTool := NewFileTool(tempDir)

	// Test invalid JSON
	_, err := fileTool.Execute(context.Background(), "invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid arguments")

	// Test unknown operation
	args := `{"type": "unknown", "path": "test.txt"}`
	_, err = fileTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown operation type")

	// Test missing required parameters
	args = `{"type": "search", "path": "."}`
	_, err = fileTool.Execute(context.Background(), args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "search pattern is required")
}

func TestFileTool_GetDefinition(t *testing.T) {
	fileTool := NewFileTool("/tmp")
	def := fileTool.GetDefinition()

	assert.Equal(t, "function", def.Type)
	assert.Equal(t, "file_operations", def.Function.Name)
	assert.Contains(t, def.Function.Description, "file system operations")

	// Check parameters
	params := def.Function.Parameters
	assert.NotNil(t, params["type"])
	assert.NotNil(t, params["path"])
	assert.NotNil(t, params["content"])
	assert.NotNil(t, params["pattern"])
	assert.NotNil(t, params["max_results"])
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1500, "1.5 KB"},
	}

	for _, tt := range tests {
		result := formatFileSize(tt.size)
		assert.Equal(t, tt.expected, result)
	}
}
