package plans

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_CreatePlan(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create manager
	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Test creating a plan
	plan, err := manager.CreatePlan("planning", "Test Plan", "A test plan", "# Test Content")
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	// Verify plan structure
	if plan.Metadata.Title != "Test Plan" {
		t.Errorf("expected title 'Test Plan', got '%s'", plan.Metadata.Title)
	}
	if plan.Metadata.Mode != "planning" {
		t.Errorf("expected mode 'planning', got '%s'", plan.Metadata.Mode)
	}
	if plan.Metadata.Description != "A test plan" {
		t.Errorf("expected description 'A test plan', got '%s'", plan.Metadata.Description)
	}
	if plan.Content != "# Test Content" {
		t.Errorf("expected content '# Test Content', got '%s'", plan.Content)
	}
	if plan.Metadata.Status != StatusDraft {
		t.Errorf("expected status '%s', got '%s'", StatusDraft, plan.Metadata.Status)
	}
	if plan.Metadata.Version != 1 {
		t.Errorf("expected version 1, got %d", plan.Metadata.Version)
	}
	if plan.Metadata.ID == "" {
		t.Error("expected plan to have an ID")
	}
	if plan.Metadata.Created.IsZero() {
		t.Error("expected plan to have creation timestamp")
	}
	if plan.Metadata.Updated.IsZero() {
		t.Error("expected plan to have update timestamp")
	}
}

func TestManager_CreatePlan_InvalidMode(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Test creating a plan with invalid mode
	_, err = manager.CreatePlan("invalid-mode", "Test Plan", "A test plan", "# Test Content")
	if err == nil {
		t.Error("expected error for invalid mode")
	}
}

func TestManager_GetPlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create a plan
	originalPlan, err := manager.CreatePlan("planning", "Test Plan", "A test plan", "# Test Content")
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	// Retrieve the plan
	retrievedPlan, err := manager.GetPlan(originalPlan.Metadata.ID)
	if err != nil {
		t.Fatalf("failed to get plan: %v", err)
	}

	// Verify the retrieved plan matches the original
	if retrievedPlan.Metadata.ID != originalPlan.Metadata.ID {
		t.Errorf("expected ID '%s', got '%s'", originalPlan.Metadata.ID, retrievedPlan.Metadata.ID)
	}
	if retrievedPlan.Metadata.Title != originalPlan.Metadata.Title {
		t.Errorf("expected title '%s', got '%s'", originalPlan.Metadata.Title, retrievedPlan.Metadata.Title)
	}
	if retrievedPlan.Content != originalPlan.Content {
		t.Errorf("expected content '%s', got '%s'", originalPlan.Content, retrievedPlan.Content)
	}
}

func TestManager_UpdatePlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create a plan
	originalPlan, err := manager.CreatePlan("planning", "Test Plan", "A test plan", "# Test Content")
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	// Update the plan
	updatedContent := "# Updated Content"
	updatedPlan, err := manager.UpdatePlan(originalPlan.Metadata.ID, updatedContent)
	if err != nil {
		t.Fatalf("failed to update plan: %v", err)
	}

	// Verify the plan was updated
	if updatedPlan.Content != updatedContent {
		t.Errorf("expected updated content '%s', got '%s'", updatedContent, updatedPlan.Content)
	}
	if updatedPlan.Metadata.Version != 2 {
		t.Errorf("expected version 2, got %d", updatedPlan.Metadata.Version)
	}
	if !updatedPlan.Metadata.Updated.After(originalPlan.Metadata.Updated) {
		t.Error("expected updated timestamp to be after original")
	}
}

func TestManager_DeletePlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create a plan
	plan, err := manager.CreatePlan("planning", "Test Plan", "A test plan", "# Test Content")
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	// Verify plan exists
	if !manager.storage.PlanExists(plan.Metadata.ID) {
		t.Error("plan should exist before deletion")
	}

	// Delete the plan
	if err := manager.DeletePlan(plan.Metadata.ID); err != nil {
		t.Fatalf("failed to delete plan: %v", err)
	}

	// Verify plan no longer exists
	if manager.storage.PlanExists(plan.Metadata.ID) {
		t.Error("plan should not exist after deletion")
	}

	// Verify getting the plan fails
	_, err = manager.GetPlan(plan.Metadata.ID)
	if err == nil {
		t.Error("expected error when getting deleted plan")
	}
}

func TestManager_ListPlans(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create multiple plans
	_, err = manager.CreatePlan("planning", "Plan 1", "First plan", "# Content 1")
	if err != nil {
		t.Fatalf("failed to create plan 1: %v", err)
	}

	_, err = manager.CreatePlan("building", "Plan 2", "Second plan", "# Content 2")
	if err != nil {
		t.Fatalf("failed to create plan 2: %v", err)
	}

	_, err = manager.CreatePlan("planning", "Plan 3", "Third plan", "# Content 3")
	if err != nil {
		t.Fatalf("failed to create plan 3: %v", err)
	}

	// List all plans
	allPlans, err := manager.ListPlans(nil)
	if err != nil {
		t.Fatalf("failed to list plans: %v", err)
	}

	if len(allPlans) != 3 {
		t.Errorf("expected 3 plans, got %d", len(allPlans))
	}

	// List plans with filter
	filter := &PlanFilter{Mode: "planning"}
	planningPlans, err := manager.ListPlans(filter)
	if err != nil {
		t.Fatalf("failed to list planning plans: %v", err)
	}

	if len(planningPlans) != 2 {
		t.Errorf("expected 2 planning plans, got %d", len(planningPlans))
	}

	// Verify the planning plans are correct
	planningTitles := make(map[string]bool)
	for _, p := range planningPlans {
		planningTitles[p.Title] = true
	}
	if !planningTitles["Plan 1"] || !planningTitles["Plan 3"] {
		t.Error("expected to find Plan 1 and Plan 3 in planning plans")
	}
}

func TestManager_SearchPlans(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create plans with different content
	_, err = manager.CreatePlan("planning", "API Design", "API design plan", "# API Design\nThis plan covers REST API design")
	if err != nil {
		t.Fatalf("failed to create API plan: %v", err)
	}

	_, err = manager.CreatePlan("building", "Database Schema", "Database implementation", "# Database\nThis covers database schema design")
	if err != nil {
		t.Fatalf("failed to create database plan: %v", err)
	}

	_, err = manager.CreatePlan("planning", "Frontend Design", "Frontend architecture", "# Frontend\nThis covers frontend design patterns")
	if err != nil {
		t.Fatalf("failed to create frontend plan: %v", err)
	}

	// Search for "API"
	apiResults, err := manager.SearchPlans("API", nil)
	if err != nil {
		t.Fatalf("failed to search for API: %v", err)
	}

	if len(apiResults) != 1 {
		t.Errorf("expected 1 API result, got %d", len(apiResults))
	}

	if apiResults[0].Title != "API Design" {
		t.Errorf("expected 'API Design', got '%s'", apiResults[0].Title)
	}

	// Search for "design" (should find multiple)
	designResults, err := manager.SearchPlans("design", nil)
	if err != nil {
		t.Fatalf("failed to search for design: %v", err)
	}

	if len(designResults) != 3 {
		t.Errorf("expected 3 design results, got %d", len(designResults))
	}
}

func TestManager_GetContext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create a plan
	plan, err := manager.CreatePlan("planning", "Test Plan", "A test plan", "# Test Content")
	if err != nil {
		t.Fatalf("failed to create plan: %v", err)
	}

	// Get context with current plan
	context, err := manager.GetContext("planning", plan.Metadata.ID)
	if err != nil {
		t.Fatalf("failed to get context: %v", err)
	}

	if context.Mode != "planning" {
		t.Errorf("expected mode 'planning', got '%s'", context.Mode)
	}
	if context.CurrentPlan == nil {
		t.Error("expected current plan to be set")
	}
	if context.CurrentPlan.Metadata.ID != plan.Metadata.ID {
		t.Errorf("expected current plan ID '%s', got '%s'", plan.Metadata.ID, context.CurrentPlan.Metadata.ID)
	}
	if len(context.RelatedPlans) != 1 {
		t.Errorf("expected 1 related plan, got %d", len(context.RelatedPlans))
	}
}

func TestManager_GetLatestPlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Create plans with different timestamps
	_, err = manager.CreatePlan("planning", "Old Plan", "Old plan", "# Old Content")
	if err != nil {
		t.Fatalf("failed to create old plan: %v", err)
	}

	// Wait a bit to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	plan2, err := manager.CreatePlan("planning", "New Plan", "New plan", "# New Content")
	if err != nil {
		t.Fatalf("failed to create new plan: %v", err)
	}

	// Get latest plan
	latestPlan, err := manager.GetLatestPlan("planning")
	if err != nil {
		t.Fatalf("failed to get latest plan: %v", err)
	}

	if latestPlan.Metadata.ID != plan2.Metadata.ID {
		t.Errorf("expected latest plan ID '%s', got '%s'", plan2.Metadata.ID, latestPlan.Metadata.ID)
	}
}

func TestManager_ValidatePlan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Test valid plan
	validPlan := &Plan{
		Metadata: PlanMetadata{
			Title:  "Valid Plan",
			Mode:   "planning",
			Status: StatusDraft,
		},
		Content: "Valid content",
	}

	if err := manager.ValidatePlan(validPlan); err != nil {
		t.Errorf("expected valid plan to pass validation: %v", err)
	}

	// Test nil plan
	if err := manager.ValidatePlan(nil); err == nil {
		t.Error("expected error for nil plan")
	}

	// Test plan without title
	invalidPlan1 := &Plan{
		Metadata: PlanMetadata{
			Mode:   "planning",
			Status: StatusDraft,
		},
		Content: "Valid content",
	}

	if err := manager.ValidatePlan(invalidPlan1); err == nil {
		t.Error("expected error for plan without title")
	}

	// Test plan with invalid mode
	invalidPlan2 := &Plan{
		Metadata: PlanMetadata{
			Title:  "Valid Plan",
			Mode:   "invalid-mode",
			Status: StatusDraft,
		},
		Content: "Valid content",
	}

	if err := manager.ValidatePlan(invalidPlan2); err == nil {
		t.Error("expected error for plan with invalid mode")
	}

	// Test plan without content
	invalidPlan3 := &Plan{
		Metadata: PlanMetadata{
			Title:  "Valid Plan",
			Mode:   "planning",
			Status: StatusDraft,
		},
		Content: "",
	}

	if err := manager.ValidatePlan(invalidPlan3); err == nil {
		t.Error("expected error for plan without content")
	}
}

func TestManager_Initialize(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "plan_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	manager := NewManager(tempDir)

	// Initialize should create the directory structure
	if err := manager.Initialize(); err != nil {
		t.Fatalf("failed to initialize: %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		"planning",
		"building",
		"debugging",
		"enhance",
		".metadata",
	}

	for _, dir := range expectedDirs {
		dirPath := filepath.Join(tempDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("expected directory '%s' to exist", dirPath)
		}
	}
}
