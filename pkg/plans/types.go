package plans

import (
	"time"
)

// Plan represents a complete plan document with metadata and content
type Plan struct {
	Metadata PlanMetadata `json:"metadata"`
	Content  string       `json:"content"`
}

// PlanMetadata contains metadata about a plan
type PlanMetadata struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Mode        string    `json:"mode"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Version     int       `json:"version"`
	Status      string    `json:"status"`
	Tags        []string  `json:"tags"`
	Author      string    `json:"author"`
}

// PlanSummary provides a lightweight view of plan metadata
type PlanSummary struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Mode        string    `json:"mode"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Updated     time.Time `json:"updated"`
	Status      string    `json:"status"`
}

// PlanFilter defines criteria for filtering plans
type PlanFilter struct {
	Mode   string    `json:"mode"`
	Status string    `json:"status"`
	Tags   []string  `json:"tags"`
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
}

// PlanTemplate defines a template for creating new plans
type PlanTemplate struct {
	Name        string `json:"name"`
	Mode        string `json:"mode"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

// PlanContext provides context information for AI calls
type PlanContext struct {
	CurrentPlan  *Plan         `json:"current_plan"`
	RelatedPlans []PlanSummary `json:"related_plans"`
	Mode         string        `json:"mode"`
}

// PlanManager defines the interface for plan management operations
type PlanManager interface {
	// Core operations
	CreatePlan(mode, title, description string, content string) (*Plan, error)
	GetPlan(id string) (*Plan, error)
	UpdatePlan(id string, content string) (*Plan, error)
	DeletePlan(id string) error

	// Listing and searching
	ListPlans(filter *PlanFilter) ([]PlanSummary, error)
	SearchPlans(query string, filter *PlanFilter) ([]PlanSummary, error)

	// Context and integration
	GetContext(mode string, planID string) (*PlanContext, error)
	GetLatestPlan(mode string) (*Plan, error)

	// Templates
	GetTemplates(mode string) ([]PlanTemplate, error)
	CreateFromTemplate(templateName, title, description string) (*Plan, error)

	// Utility
	Initialize() error
	ValidatePlan(plan *Plan) error
}

// PlanStatus constants
const (
	StatusDraft     = "draft"
	StatusActive    = "active"
	StatusCompleted = "completed"
	StatusArchived  = "archived"
)

// SupportedModes contains the list of supported plan modes
var SupportedModes = []string{"planning", "building", "debugging", "enhance"}
