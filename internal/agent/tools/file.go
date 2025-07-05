package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hammie/rubrduck/internal/ai"
	"github.com/rs/zerolog/log"
)

// FileTool provides file system operations
type FileTool struct {
	basePath string
}

// NewFileTool creates a new file tool instance
func NewFileTool(basePath string) *FileTool {
	return &FileTool{
		basePath: basePath,
	}
}

// GetDefinition returns the tool definition for the AI
func (f *FileTool) GetDefinition() ai.Tool {
	return ai.Tool{
		Type: "function",
		Function: ai.ToolFunction{
			Name:        "file_operations",
			Description: "Perform file system operations including read, write, list, and search",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"read", "write", "list", "search", "append"},
						"description": "The type of file operation to perform",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The file or directory path (relative to project root)",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Content to write to file (only for write operations)",
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "Search pattern for file search (only for search operations)",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return (for list and search operations)",
						"default":     50,
					},
				},
				"required": []string{"type", "path"},
			},
		},
	}
}

// Execute runs the file operation with the given arguments
func (f *FileTool) Execute(ctx context.Context, args string) (string, error) {
	var params struct {
		Type       string `json:"type"`
		Path       string `json:"path"`
		Content    string `json:"content"`
		Pattern    string `json:"pattern"`
		MaxResults int    `json:"max_results"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Set default max results
	if params.MaxResults == 0 {
		params.MaxResults = 50
	}

	// Validate and sanitize path
	fullPath, err := f.sanitizePath(params.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	switch params.Type {
	case "read":
		return f.readFile(fullPath)
	case "write":
		return f.writeFile(fullPath, params.Content)
	case "append":
		return f.appendFile(fullPath, params.Content)
	case "list":
		return f.listDirectory(fullPath, params.MaxResults)
	case "search":
		return f.searchFiles(fullPath, params.Pattern, params.MaxResults)
	default:
		return "", fmt.Errorf("unknown operation type: %s", params.Type)
	}
}

// sanitizePath ensures the path is safe and within the project bounds
func (f *FileTool) sanitizePath(path string) (string, error) {
	if path == "" {
		return f.basePath, nil
	}

	// Clean the path to remove any .. or . components
	cleanPath := filepath.Clean(path)

	// If it's an absolute path, make it relative to base
	if filepath.IsAbs(cleanPath) {
		relPath, err := filepath.Rel(f.basePath, cleanPath)
		if err != nil {
			return "", fmt.Errorf("path outside project bounds")
		}
		cleanPath = relPath
	}

	// Join with base path
	fullPath := filepath.Join(f.basePath, cleanPath)

	// Ensure the final path is within the base path
	relPath, err := filepath.Rel(f.basePath, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path outside project bounds")
	}

	return fullPath, nil
}

// readFile reads the contents of a file
func (f *FileTool) readFile(path string) (string, error) {
	log.Debug().Str("path", path).Msg("Reading file")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Check file size to avoid reading huge files
	if len(content) > 1024*1024 { // 1MB limit
		return fmt.Sprintf("File too large (%d bytes). Showing first 1KB:\n\n%s",
			len(content), string(content[:1024])), nil
	}

	return string(content), nil
}

// writeFile writes content to a file
func (f *FileTool) writeFile(path, content string) (string, error) {
	log.Debug().
		Str("path", path).
		Int("content_size", len(content)).
		Msg("Writing file")

	// Check for very large files that might cause streaming timeouts
	const largeFileThreshold = 50 * 1024      // 50KB
	const veryLargeFileThreshold = 200 * 1024 // 200KB

	if len(content) > veryLargeFileThreshold {
		log.Error().
			Str("path", path).
			Int("size_bytes", len(content)).
			Int("size_kb", len(content)/1024).
			Msg("File content is extremely large - this may cause timeout issues during streaming")

		// For very large files, suggest using incremental updates
		return "", fmt.Errorf("file content is too large (%d KB) for a single write operation. Consider breaking this into smaller incremental updates or using append operations", len(content)/1024)
	}

	// Log warning for large files
	if len(content) > largeFileThreshold {
		log.Warn().
			Str("path", path).
			Int("size_bytes", len(content)).
			Int("size_kb", len(content)/1024).
			Msg("Writing large file - this may take time")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists and is read-only
	if _, err := os.Stat(path); err == nil {
		// File exists, check if it's read-only
		info, err := os.Stat(path)
		if err == nil && info.Mode()&0200 == 0 {
			return "", fmt.Errorf("file is read-only")
		}
	}

	// Write file with progress tracking for large files
	startTime := time.Now()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	duration := time.Since(startTime)

	// Log completion with timing for large files
	if len(content) > largeFileThreshold {
		log.Info().
			Str("path", path).
			Int("size_bytes", len(content)).
			Dur("duration", duration).
			Float64("kb_per_second", float64(len(content))/1024/duration.Seconds()).
			Msg("Large file write completed")
	}

	// Provide detailed feedback including write speed for large files
	if len(content) > largeFileThreshold {
		return fmt.Sprintf("Successfully wrote %d KB to %s (took %v, %.1f KB/s)",
			len(content)/1024, path, duration, float64(len(content))/1024/duration.Seconds()), nil
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s (took %v)", len(content), path, duration), nil
}

// appendFile appends content to a file
func (f *FileTool) appendFile(path, content string) (string, error) {
	log.Debug().
		Str("path", path).
		Int("content_size", len(content)).
		Msg("Appending to file")

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Check if file exists and is read-only
	if _, err := os.Stat(path); err == nil {
		// File exists, check if it's read-only
		info, err := os.Stat(path)
		if err == nil && info.Mode()&0200 == 0 {
			return "", fmt.Errorf("file is read-only")
		}
	}

	// Open file in append mode
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file for append: %w", err)
	}
	defer file.Close()

	// Append content
	startTime := time.Now()
	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to append to file: %w", err)
	}
	duration := time.Since(startTime)

	// Use same thresholds as writeFile
	const largeFileThreshold = 50 * 1024 // 50KB

	// Log completion with timing for large files
	if len(content) > largeFileThreshold {
		log.Info().
			Str("path", path).
			Int("size_bytes", len(content)).
			Dur("duration", duration).
			Float64("kb_per_second", float64(len(content))/1024/duration.Seconds()).
			Msg("Large file append completed")
	}

	// Provide detailed feedback including write speed for large files
	if len(content) > largeFileThreshold {
		return fmt.Sprintf("Successfully appended %d KB to %s (took %v, %.1f KB/s)",
			len(content)/1024, path, duration, float64(len(content))/1024/duration.Seconds()), nil
	}

	return fmt.Sprintf("Successfully appended %d bytes to %s (took %v)", len(content), path, duration), nil
}

// listDirectory lists the contents of a directory
func (f *FileTool) listDirectory(path string, maxResults int) (string, error) {
	log.Debug().Str("path", path).Msg("Listing directory")

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Contents of %s:\n\n", path))

	count := 0
	for _, entry := range entries {
		if count >= maxResults {
			result.WriteString(fmt.Sprintf("\n... and %d more entries", len(entries)-maxResults))
			break
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Format file info
		var size string
		if entry.IsDir() {
			size = "<DIR>"
		} else {
			size = formatFileSize(info.Size())
		}

		result.WriteString(fmt.Sprintf("%-40s %8s %s\n",
			entry.Name(), size, info.Mode().String()))
		count++
	}

	return result.String(), nil
}

// searchFiles searches for files matching a pattern
func (f *FileTool) searchFiles(basePath, pattern string, maxResults int) (string, error) {
	log.Debug().Str("basePath", basePath).Str("pattern", pattern).Msg("Searching files")

	if pattern == "" {
		return "", fmt.Errorf("search pattern is required")
	}

	var results []string
	stopErr := errors.New("stop search")
	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches pattern
		if !d.IsDir() && strings.Contains(strings.ToLower(filepath.Base(path)),
			strings.ToLower(pattern)) {
			relPath, _ := filepath.Rel(basePath, path)
			results = append(results, relPath)
		}

		// Stop if we have enough results
		if len(results) >= maxResults {
			return stopErr
		}

		return nil
	})

	if err != nil && err == stopErr {
		err = nil
	}
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return fmt.Sprintf("No files found matching pattern '%s' in %s", pattern, basePath), nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d files matching '%s':\n\n", len(results), pattern))

	for _, file := range results {
		result.WriteString(fmt.Sprintf("- %s\n", file))
	}

	return result.String(), nil
}

// formatFileSize formats file size in human readable format
func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
