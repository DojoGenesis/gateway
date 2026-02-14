package agent

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// ErrAgentNotFound is returned when a requested agent does not exist.
var ErrAgentNotFound = errors.New("agent not found")

type AgentManager struct {
	db *sql.DB
}

func NewAgentManager(db *sql.DB) (*AgentManager, error) {
	return &AgentManager{db: db}, nil
}

func validateAgentType(agentType string) error {
	validTypes := map[string]bool{
		"primary":    true,
		"specialist": true,
		"utility":    true,
	}
	if !validTypes[agentType] {
		return fmt.Errorf("invalid agent type: must be one of 'primary', 'specialist', or 'utility'")
	}
	return nil
}

func validateAgentStatus(status string) error {
	validStatuses := map[string]bool{
		"active":       true,
		"inactive":     true,
		"experimental": true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid agent status: must be one of 'active', 'inactive', or 'experimental'")
	}
	return nil
}

func validateCapabilityType(capType string) error {
	validTypes := map[string]bool{
		"tool":  true,
		"skill": true,
		"model": true,
	}
	if !validTypes[capType] {
		return fmt.Errorf("invalid capability type: must be one of 'tool', 'skill', or 'model'")
	}
	return nil
}

func (am *AgentManager) RegisterAgent(ctx context.Context, req RegisterAgentRequest) (*Agent, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("agent name is required")
	}

	if req.ID == "" {
		return nil, fmt.Errorf("agent ID is required")
	}

	if req.Description == "" {
		return nil, fmt.Errorf("agent description is required")
	}

	if err := validateAgentType(req.Type); err != nil {
		return nil, err
	}

	if req.Status == "" {
		req.Status = "active"
	}

	if err := validateAgentStatus(req.Status); err != nil {
		return nil, err
	}

	tx, err := am.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	agent := &Agent{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      req.Status,
		ModelName:   req.ModelName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	query := `
		INSERT INTO agents (id, name, description, type, status, model_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			type = excluded.type,
			status = excluded.status,
			model_name = excluded.model_name,
			updated_at = excluded.updated_at
	`

	_, err = tx.ExecContext(ctx, query,
		agent.ID,
		agent.Name,
		agent.Description,
		agent.Type,
		agent.Status,
		agent.ModelName,
		agent.CreatedAt,
		agent.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to register agent: %w", err)
	}

	for _, cap := range req.Capabilities {
		if err := validateCapabilityType(cap.CapabilityType); err != nil {
			return nil, err
		}

		capID := uuid.New().String()
		capQuery := `
			INSERT INTO agent_capabilities (id, agent_id, capability_type, name, description, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(agent_id, capability_type, name) DO UPDATE SET
				description = excluded.description
		`

		_, err = tx.ExecContext(ctx, capQuery,
			capID,
			agent.ID,
			cap.CapabilityType,
			cap.Name,
			cap.Description,
			now,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to register capability: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return agent, nil
}

func (am *AgentManager) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	query := `SELECT id, name, description, type, status, model_name, created_at, updated_at
	          FROM agents WHERE id = ?`

	var agent Agent
	var modelName sql.NullString

	err := am.db.QueryRowContext(ctx, query, agentID).Scan(
		&agent.ID,
		&agent.Name,
		&agent.Description,
		&agent.Type,
		&agent.Status,
		&modelName,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrAgentNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve agent: %w", err)
	}

	if modelName.Valid {
		agent.ModelName = modelName.String
	}

	return &agent, nil
}

func (am *AgentManager) ListAgents(ctx context.Context) ([]Agent, error) {
	query := `SELECT id, name, description, type, status, model_name, created_at, updated_at
	          FROM agents ORDER BY created_at DESC`

	rows, err := am.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	agents := []Agent{}
	for rows.Next() {
		var agent Agent
		var modelName sql.NullString

		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Description,
			&agent.Type,
			&agent.Status,
			&modelName,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan agent row: %w", err)
		}

		if modelName.Valid {
			agent.ModelName = modelName.String
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

func (am *AgentManager) ListAgentsPaginated(ctx context.Context, page, limit int) (*PaginatedAgentsResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM agents`
	err := am.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count agents: %w", err)
	}

	offset := (page - 1) * limit
	query := `SELECT id, name, description, type, status, model_name, created_at, updated_at
	          FROM agents ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := am.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}
	defer rows.Close()

	agents := []Agent{}
	for rows.Next() {
		var agent Agent
		var modelName sql.NullString

		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Description,
			&agent.Type,
			&agent.Status,
			&modelName,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan agent row: %w", err)
		}

		if modelName.Valid {
			agent.ModelName = modelName.String
		}

		agents = append(agents, agent)
	}

	totalPages := (total + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	return &PaginatedAgentsResponse{
		Agents: agents,
		Pagination: PaginationMetadata{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}, nil
}

func (am *AgentManager) GetAgentCapabilities(ctx context.Context, agentID string) ([]AgentCapability, error) {
	var exists bool
	err := am.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM agents WHERE id = ?)", agentID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check agent existence: %w", err)
	}
	if !exists {
		return nil, ErrAgentNotFound
	}

	query := `SELECT id, agent_id, capability_type, name, description, created_at
	          FROM agent_capabilities WHERE agent_id = ? ORDER BY capability_type, name`

	rows, err := am.db.QueryContext(ctx, query, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent capabilities: %w", err)
	}
	defer rows.Close()

	capabilities := []AgentCapability{}
	for rows.Next() {
		var cap AgentCapability
		var description sql.NullString

		err := rows.Scan(
			&cap.ID,
			&cap.AgentID,
			&cap.CapabilityType,
			&cap.Name,
			&description,
			&cap.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan capability row: %w", err)
		}

		if description.Valid {
			cap.Description = description.String
		}

		capabilities = append(capabilities, cap)
	}

	return capabilities, nil
}
