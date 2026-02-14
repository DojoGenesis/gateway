package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Plan struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Goals       []string               `json:"goals"`
	Milestones  []Milestone            `json:"milestones"`
	Progress    float64                `json:"progress"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type Milestone struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	DueDate     time.Time              `json:"due_date,omitempty"`
	CompletedAt time.Time              `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusCancelled  = "cancelled"
)

var plansDirectoryOverride string

func getPlansDirectory() string {
	if plansDirectoryOverride != "" {
		return plansDirectoryOverride
	}

	// Use DOJO_DATA_DIR if set, otherwise stable home-directory location
	if dataDir := os.Getenv("DOJO_DATA_DIR"); dataDir != "" {
		return filepath.Join(dataDir, "planning")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".dojo/planning"
	}

	// Tauri sidecar mode: use ~/.dojo-genesis/planning
	if os.Getenv("DOJO_BINARIES_DIR") != "" {
		return filepath.Join(homeDir, ".dojo-genesis", "planning")
	}

	return filepath.Join(homeDir, ".dojo", "planning")
}

func ensurePlansDirectory() error {
	plansDir := getPlansDirectory()
	return os.MkdirAll(plansDir, 0755)
}

func getPlanFilePath(planID string) string {
	return filepath.Join(getPlansDirectory(), fmt.Sprintf("%s.json", planID))
}

func savePlan(plan Plan) error {
	if err := ensurePlansDirectory(); err != nil {
		return fmt.Errorf("failed to create plans directory: %w", err)
	}

	planPath := getPlanFilePath(plan.ID)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(planPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

func loadPlan(planID string) (*Plan, error) {
	planPath := getPlanFilePath(planID)
	data, err := os.ReadFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plan not found: %s", planID)
		}
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

func CreatePlan(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	title, ok := params["title"].(string)
	if !ok || title == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "title parameter is required",
		}, nil
	}

	description := GetStringParam(params, "description", "")
	goals := GetStringSliceParam(params, "goals", []string{})
	metadata := GetMapParam(params, "metadata", make(map[string]interface{}))

	plan := Plan{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Goals:       goals,
		Milestones:  []Milestone{},
		Progress:    0.0,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    metadata,
	}

	if err := savePlan(plan); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to save plan: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"plan":    plan,
		"message": fmt.Sprintf("Plan '%s' created successfully", plan.Title),
	}, nil
}

func UpdatePlan(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	planID, ok := params["plan_id"].(string)
	if !ok || planID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "plan_id parameter is required",
		}, nil
	}

	plan, err := loadPlan(planID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	if title := GetStringParam(params, "title", ""); title != "" {
		plan.Title = title
	}

	if description := GetStringParam(params, "description", ""); description != "" {
		plan.Description = description
	}

	if goals, ok := params["goals"].([]interface{}); ok {
		plan.Goals = make([]string, 0, len(goals))
		for _, g := range goals {
			if goalStr, ok := g.(string); ok {
				plan.Goals = append(plan.Goals, goalStr)
			}
		}
	}

	if status := GetStringParam(params, "status", ""); status != "" {
		validStatuses := map[string]bool{
			StatusPending:    true,
			StatusInProgress: true,
			StatusCompleted:  true,
			StatusCancelled:  true,
		}
		if !validStatuses[status] {
			return map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("invalid status: %s (must be one of: pending, in_progress, completed, cancelled)", status),
			}, nil
		}
		plan.Status = status
	}

	if metadata, ok := params["metadata"].(map[string]interface{}); ok {
		for k, v := range metadata {
			plan.Metadata[k] = v
		}
	}

	plan.UpdatedAt = time.Now()

	if err := savePlan(*plan); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to save plan: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success": true,
		"plan":    plan,
		"message": fmt.Sprintf("Plan '%s' updated successfully", plan.Title),
	}, nil
}

func TrackProgress(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	planID, ok := params["plan_id"].(string)
	if !ok || planID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "plan_id parameter is required",
		}, nil
	}

	plan, err := loadPlan(planID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	if len(plan.Milestones) == 0 {
		return map[string]interface{}{
			"success":  true,
			"plan_id":  plan.ID,
			"progress": 0.0,
			"message":  "No milestones to track",
		}, nil
	}

	completedMilestones := 0
	inProgressMilestones := 0
	for _, m := range plan.Milestones {
		if m.Status == StatusCompleted {
			completedMilestones++
		} else if m.Status == StatusInProgress {
			inProgressMilestones++
		}
	}

	progress := float64(completedMilestones) / float64(len(plan.Milestones)) * 100.0
	plan.Progress = progress
	plan.UpdatedAt = time.Now()

	if progress == 100.0 && plan.Status != StatusCompleted {
		plan.Status = StatusCompleted
	} else if progress > 0 && plan.Status == StatusPending {
		plan.Status = StatusInProgress
	}

	if err := savePlan(*plan); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to save plan: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":                true,
		"plan_id":                plan.ID,
		"progress":               progress,
		"total_milestones":       len(plan.Milestones),
		"completed_milestones":   completedMilestones,
		"in_progress_milestones": inProgressMilestones,
		"status":                 plan.Status,
		"message":                fmt.Sprintf("Progress: %.1f%% (%d/%d milestones completed)", progress, completedMilestones, len(plan.Milestones)),
	}, nil
}

func ValidatePlan(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	planID, ok := params["plan_id"].(string)
	if !ok || planID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "plan_id parameter is required",
		}, nil
	}

	plan, err := loadPlan(planID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	issues := []string{}
	warnings := []string{}

	if plan.Title == "" {
		issues = append(issues, "Plan has no title")
	}

	if len(plan.Goals) == 0 {
		warnings = append(warnings, "Plan has no goals defined")
	}

	if len(plan.Milestones) == 0 {
		warnings = append(warnings, "Plan has no milestones defined")
	}

	duplicateMilestones := make(map[string]int)
	for _, m := range plan.Milestones {
		if m.Title == "" {
			issues = append(issues, fmt.Sprintf("Milestone %s has no title", m.ID))
		}
		duplicateMilestones[m.Title]++
	}

	for title, count := range duplicateMilestones {
		if count > 1 {
			warnings = append(warnings, fmt.Sprintf("Duplicate milestone title: '%s' (appears %d times)", title, count))
		}
	}

	now := time.Now()
	overdueMilestones := 0
	for _, m := range plan.Milestones {
		if !m.DueDate.IsZero() && m.DueDate.Before(now) && m.Status != StatusCompleted {
			overdueMilestones++
		}
	}

	if overdueMilestones > 0 {
		warnings = append(warnings, fmt.Sprintf("%d milestone(s) are overdue", overdueMilestones))
	}

	isValid := len(issues) == 0

	return map[string]interface{}{
		"success":  true,
		"valid":    isValid,
		"plan_id":  plan.ID,
		"issues":   issues,
		"warnings": warnings,
		"message":  fmt.Sprintf("Plan validation: %d issues, %d warnings", len(issues), len(warnings)),
	}, nil
}

func CreateMilestone(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	planID, ok := params["plan_id"].(string)
	if !ok || planID == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "plan_id parameter is required",
		}, nil
	}

	title, ok := params["title"].(string)
	if !ok || title == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "title parameter is required",
		}, nil
	}

	plan, err := loadPlan(planID)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}, nil
	}

	description := GetStringParam(params, "description", "")
	status := GetStringParam(params, "status", StatusPending)
	metadata := GetMapParam(params, "metadata", make(map[string]interface{}))

	validStatuses := map[string]bool{
		StatusPending:    true,
		StatusInProgress: true,
		StatusCompleted:  true,
		StatusCancelled:  true,
	}
	if !validStatuses[status] {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("invalid status: %s (must be one of: pending, in_progress, completed, cancelled)", status),
		}, nil
	}

	milestone := Milestone{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Status:      status,
		Metadata:    metadata,
	}

	if dueDateStr := GetStringParam(params, "due_date", ""); dueDateStr != "" {
		dueDate, err := time.Parse(time.RFC3339, dueDateStr)
		if err != nil {
			dueDate, err = time.Parse("2006-01-02", dueDateStr)
			if err != nil {
				return map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("invalid due_date format: %v (use RFC3339 or YYYY-MM-DD)", err),
				}, nil
			}
		}
		milestone.DueDate = dueDate
	}

	if status == StatusCompleted {
		milestone.CompletedAt = time.Now()
	}

	plan.Milestones = append(plan.Milestones, milestone)
	plan.UpdatedAt = time.Now()

	if err := savePlan(*plan); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("failed to save plan: %v", err),
		}, nil
	}

	return map[string]interface{}{
		"success":   true,
		"milestone": milestone,
		"plan_id":   plan.ID,
		"message":   fmt.Sprintf("Milestone '%s' created successfully", milestone.Title),
	}, nil
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "create_plan",
		Description: "Create a new structured plan with goals and milestones",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"title": map[string]interface{}{
					"type":        "string",
					"description": "The title of the plan",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "A detailed description of the plan",
				},
				"goals": map[string]interface{}{
					"type":        "array",
					"description": "List of goals for the plan",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"metadata": map[string]interface{}{
					"type":        "object",
					"description": "Additional metadata for the plan",
				},
			},
			"required": []string{"title"},
		},
		Function: CreatePlan,
	})

	RegisterTool(&ToolDefinition{
		Name:        "update_plan",
		Description: "Update an existing plan's properties",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"plan_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the plan to update",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "New title for the plan",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "New description for the plan",
				},
				"goals": map[string]interface{}{
					"type":        "array",
					"description": "Updated list of goals",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "New status (pending, in_progress, completed, cancelled)",
					"enum":        []string{StatusPending, StatusInProgress, StatusCompleted, StatusCancelled},
				},
				"metadata": map[string]interface{}{
					"type":        "object",
					"description": "Additional metadata to merge with existing",
				},
			},
			"required": []string{"plan_id"},
		},
		Function: UpdatePlan,
	})

	RegisterTool(&ToolDefinition{
		Name:        "track_progress",
		Description: "Track and calculate progress of a plan based on milestone completion",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"plan_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the plan to track",
				},
			},
			"required": []string{"plan_id"},
		},
		Function: TrackProgress,
	})

	RegisterTool(&ToolDefinition{
		Name:        "validate_plan",
		Description: "Validate a plan for completeness and consistency",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"plan_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the plan to validate",
				},
			},
			"required": []string{"plan_id"},
		},
		Function: ValidatePlan,
	})

	RegisterTool(&ToolDefinition{
		Name:        "create_milestone",
		Description: "Create a new milestone for a plan",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"plan_id": map[string]interface{}{
					"type":        "string",
					"description": "The ID of the plan",
				},
				"title": map[string]interface{}{
					"type":        "string",
					"description": "The title of the milestone",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "A detailed description of the milestone",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "Status of the milestone (pending, in_progress, completed, cancelled)",
					"enum":        []string{StatusPending, StatusInProgress, StatusCompleted, StatusCancelled},
				},
				"due_date": map[string]interface{}{
					"type":        "string",
					"description": "Due date in RFC3339 or YYYY-MM-DD format",
				},
				"metadata": map[string]interface{}{
					"type":        "object",
					"description": "Additional metadata for the milestone",
				},
			},
			"required": []string{"plan_id", "title"},
		},
		Function: CreateMilestone,
	})
}
