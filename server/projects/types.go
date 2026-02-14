package projects

import "time"

type Project struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	TemplateID     string                 `json:"template_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	LastAccessedAt time.Time              `json:"last_accessed_at"`
	Settings       map[string]interface{} `json:"settings"`
	Metadata       map[string]interface{} `json:"metadata"`
	Status         string                 `json:"status"`
}

type ProjectTemplate struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Structure       map[string]interface{} `json:"structure"`
	DefaultSettings map[string]interface{} `json:"default_settings"`
	IsSystem        bool                   `json:"is_system"`
	CreatedAt       time.Time              `json:"created_at"`
}

const (
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusDeleted  = "deleted"
)

const (
	CategoryResearch    = "research"
	CategoryDevelopment = "development"
	CategoryDesign      = "design"
	CategoryAnalysis    = "analysis"
)
