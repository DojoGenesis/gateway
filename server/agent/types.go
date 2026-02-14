package agent

import "time"

// QueryType represents the classification category for user queries.
// It determines whether a query should be handled with a template response
// or routed to the Primary Agent for complex reasoning.
type QueryType int

const (
	// Simple indicates queries that can be handled with template responses.
	// Examples: greetings, simple questions, help requests.
	Simple QueryType = iota

	// Complex indicates queries requiring agent reasoning and tool usage.
	// Examples: code generation, debugging, system design.
	Complex
)

// String returns the string representation of QueryType.
func (qt QueryType) String() string {
	switch qt {
	case Simple:
		return "Simple"
	case Complex:
		return "Complex"
	default:
		return "Unknown"
	}
}

// ClassificationResult contains the query type and confidence score
type ClassificationResult struct {
	Type       QueryType
	Confidence float64 // 0.0 to 1.0
	Reasoning  string  // Human-readable explanation
}

type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	ModelName   string    `json:"model_name,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AgentCapability struct {
	ID             string    `json:"id"`
	AgentID        string    `json:"agent_id"`
	CapabilityType string    `json:"capability_type"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type RegisterAgentRequest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	ModelName    string            `json:"model_name,omitempty"`
	Capabilities []AgentCapability `json:"capabilities,omitempty"`
}

type PaginationRequest struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

type PaginationMetadata struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

type PaginatedAgentsResponse struct {
	Agents     []Agent            `json:"agents"`
	Pagination PaginationMetadata `json:"pagination"`
}
