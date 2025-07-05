package plans

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FileStorage implements plan storage using the file system
type FileStorage struct {
	baseDir string
}

// NewFileStorage creates a new file-based storage manager
func NewFileStorage(baseDir string) *FileStorage {
	return &FileStorage{
		baseDir: baseDir,
	}
}

// Initialize creates the necessary directory structure
func (fs *FileStorage) Initialize() error {
	// Create base .duckie directory
	if err := os.MkdirAll(fs.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create metadata directory for plan metadata and content
	metadataDir := filepath.Join(fs.baseDir, ".metadata")
	if err := os.MkdirAll(metadataDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	return nil
}

// planPath returns the file path for a plan
func (fs *FileStorage) planPath(id string) string {
	return filepath.Join(fs.baseDir, ".metadata", fmt.Sprintf("%s.json", id))
}

// contentPath returns the file path for plan content
func (fs *FileStorage) contentPath(id string) string {
	return filepath.Join(fs.baseDir, ".metadata", fmt.Sprintf("%s.md", id))
}

// SavePlan saves a plan to the file system
func (fs *FileStorage) SavePlan(plan *Plan) error {
	if plan == nil {
		return fmt.Errorf("plan cannot be nil")
	}

	// Ensure plan has an ID
	if plan.Metadata.ID == "" {
		plan.Metadata.ID = uuid.New().String()
	}

	// Update timestamps
	now := time.Now()
	if plan.Metadata.Created.IsZero() {
		plan.Metadata.Created = now
	}
	plan.Metadata.Updated = now

	// Save metadata
	metadataPath := fs.planPath(plan.Metadata.ID)
	metadataData, err := json.MarshalIndent(plan.Metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	// Save content
	contentPath := fs.contentPath(plan.Metadata.ID)
	if err := os.WriteFile(contentPath, []byte(plan.Content), 0644); err != nil {
		return fmt.Errorf("failed to write content: %w", err)
	}

	return nil
}

// LoadPlan loads a plan from the file system
func (fs *FileStorage) LoadPlan(id string) (*Plan, error) {
	// Load metadata
	metadataPath := fs.planPath(id)
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata PlanMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	// Load content
	contentPath := fs.contentPath(id)
	contentData, err := os.ReadFile(contentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return &Plan{
		Metadata: metadata,
		Content:  string(contentData),
	}, nil
}

// DeletePlan removes a plan from the file system
func (fs *FileStorage) DeletePlan(id string) error {
	// Remove metadata file
	metadataPath := fs.planPath(id)
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove metadata: %w", err)
	}

	// Remove content file
	contentPath := fs.contentPath(id)
	if err := os.Remove(contentPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove content: %w", err)
	}

	return nil
}

// ListPlans returns all plan summaries
func (fs *FileStorage) ListPlans() ([]PlanSummary, error) {
	metadataDir := filepath.Join(fs.baseDir, ".metadata")

	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PlanSummary{}, nil
		}
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var summaries []PlanSummary
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Extract ID from filename
		id := strings.TrimSuffix(entry.Name(), ".json")

		// Load metadata
		plan, err := fs.LoadPlan(id)
		if err != nil {
			// Skip corrupted plans
			continue
		}

		summaries = append(summaries, PlanSummary{
			ID:          plan.Metadata.ID,
			Title:       plan.Metadata.Title,
			Mode:        plan.Metadata.Mode,
			Description: plan.Metadata.Description,
			Created:     plan.Metadata.Created,
			Updated:     plan.Metadata.Updated,
			Status:      plan.Metadata.Status,
		})
	}

	return summaries, nil
}

// GetLatestPlanByMode returns the most recent plan for a given mode
func (fs *FileStorage) GetLatestPlanByMode(mode string) (*Plan, error) {
	summaries, err := fs.ListPlans()
	if err != nil {
		return nil, err
	}

	var latestPlan *Plan
	var latestTime time.Time

	for _, summary := range summaries {
		if summary.Mode == mode && summary.Updated.After(latestTime) {
			plan, err := fs.LoadPlan(summary.ID)
			if err != nil {
				continue
			}
			latestPlan = plan
			latestTime = summary.Updated
		}
	}

	if latestPlan == nil {
		return nil, fmt.Errorf("no plans found for mode: %s", mode)
	}

	return latestPlan, nil
}

// PlanExists checks if a plan exists
func (fs *FileStorage) PlanExists(id string) bool {
	metadataPath := fs.planPath(id)
	_, err := os.Stat(metadataPath)
	return err == nil
}
