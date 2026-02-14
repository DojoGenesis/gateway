package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestPlansDir(t *testing.T) string {
	tempDir := t.TempDir()
	plansDir := filepath.Join(tempDir, ".dojo", "planning")
	if err := os.MkdirAll(plansDir, 0755); err != nil {
		t.Fatalf("Failed to create test plans directory: %v", err)
	}

	oldPlansDir := plansDirectoryOverride
	plansDirectoryOverride = plansDir
	t.Cleanup(func() {
		plansDirectoryOverride = oldPlansDir
	})

	return plansDir
}

func TestCreatePlan(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		params      map[string]interface{}
		expectError bool
		validate    func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "create plan with all fields",
			params: map[string]interface{}{
				"title":       "Test Plan",
				"description": "A test plan for unit testing",
				"goals":       []string{"Goal 1", "Goal 2", "Goal 3"},
				"metadata": map[string]interface{}{
					"author":   "test",
					"priority": "high",
				},
			},
			expectError: false,
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true, got %v", result["success"])
				}
				plan := result["plan"].(Plan)
				if plan.Title != "Test Plan" {
					t.Errorf("Expected title 'Test Plan', got '%s'", plan.Title)
				}
				if plan.Description != "A test plan for unit testing" {
					t.Errorf("Expected description to match")
				}
				if len(plan.Goals) != 3 {
					t.Errorf("Expected 3 goals, got %d", len(plan.Goals))
				}
				if plan.Status != StatusPending {
					t.Errorf("Expected status 'pending', got '%s'", plan.Status)
				}
				if plan.Progress != 0.0 {
					t.Errorf("Expected progress 0.0, got %f", plan.Progress)
				}
				if plan.Metadata["author"] != "test" {
					t.Errorf("Expected metadata author 'test'")
				}
			},
		},
		{
			name: "create plan with minimal fields",
			params: map[string]interface{}{
				"title": "Minimal Plan",
			},
			expectError: false,
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				plan := result["plan"].(Plan)
				if plan.Title != "Minimal Plan" {
					t.Errorf("Expected title 'Minimal Plan'")
				}
				if len(plan.Goals) != 0 {
					t.Errorf("Expected 0 goals")
				}
				if len(plan.Milestones) != 0 {
					t.Errorf("Expected 0 milestones")
				}
			},
		},
		{
			name:        "create plan without title",
			params:      map[string]interface{}{},
			expectError: true,
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["success"].(bool) {
					t.Errorf("Expected success=false")
				}
				if result["error"] == nil {
					t.Errorf("Expected error message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreatePlan(ctx, tt.params)
			if err != nil {
				t.Fatalf("CreatePlan returned error: %v", err)
			}

			if tt.expectError {
				if result["success"].(bool) {
					t.Errorf("Expected error but got success")
				}
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestUpdatePlan(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	createResult, _ := CreatePlan(ctx, map[string]interface{}{
		"title":       "Original Plan",
		"description": "Original description",
		"goals":       []string{"Goal 1"},
	})
	plan := createResult["plan"].(Plan)

	tests := []struct {
		name     string
		params   map[string]interface{}
		validate func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "update title",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"title":   "Updated Plan",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				updatedPlan := result["plan"].(*Plan)
				if updatedPlan.Title != "Updated Plan" {
					t.Errorf("Expected title 'Updated Plan', got '%s'", updatedPlan.Title)
				}
			},
		},
		{
			name: "update status",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"status":  StatusInProgress,
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				updatedPlan := result["plan"].(*Plan)
				if updatedPlan.Status != StatusInProgress {
					t.Errorf("Expected status 'in_progress', got '%s'", updatedPlan.Status)
				}
			},
		},
		{
			name: "update goals",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"goals":   []interface{}{"New Goal 1", "New Goal 2"},
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				updatedPlan := result["plan"].(*Plan)
				if len(updatedPlan.Goals) != 2 {
					t.Errorf("Expected 2 goals, got %d", len(updatedPlan.Goals))
				}
			},
		},
		{
			name: "invalid status",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"status":  "invalid_status",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["success"].(bool) {
					t.Errorf("Expected success=false for invalid status")
				}
			},
		},
		{
			name: "non-existent plan",
			params: map[string]interface{}{
				"plan_id": "non-existent-id",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["success"].(bool) {
					t.Errorf("Expected success=false for non-existent plan")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UpdatePlan(ctx, tt.params)
			if err != nil {
				t.Fatalf("UpdatePlan returned error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCreateMilestone(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	createResult, _ := CreatePlan(ctx, map[string]interface{}{
		"title": "Test Plan for Milestones",
	})
	plan := createResult["plan"].(Plan)

	tests := []struct {
		name     string
		params   map[string]interface{}
		validate func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "create milestone with all fields",
			params: map[string]interface{}{
				"plan_id":     plan.ID,
				"title":       "Milestone 1",
				"description": "First milestone",
				"status":      StatusPending,
				"due_date":    "2024-12-31",
				"metadata": map[string]interface{}{
					"priority": "high",
				},
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true, got error: %v", result["error"])
				}
				milestone := result["milestone"].(Milestone)
				if milestone.Title != "Milestone 1" {
					t.Errorf("Expected title 'Milestone 1', got '%s'", milestone.Title)
				}
				if milestone.Status != StatusPending {
					t.Errorf("Expected status 'pending', got '%s'", milestone.Status)
				}
				if milestone.DueDate.IsZero() {
					t.Errorf("Expected due_date to be set")
				}
			},
		},
		{
			name: "create milestone with minimal fields",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"title":   "Minimal Milestone",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				milestone := result["milestone"].(Milestone)
				if milestone.Status != StatusPending {
					t.Errorf("Expected default status 'pending', got '%s'", milestone.Status)
				}
			},
		},
		{
			name: "create milestone with RFC3339 date",
			params: map[string]interface{}{
				"plan_id":  plan.ID,
				"title":    "Milestone with RFC3339",
				"due_date": "2024-12-31T23:59:59Z",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				milestone := result["milestone"].(Milestone)
				if milestone.DueDate.IsZero() {
					t.Errorf("Expected due_date to be parsed")
				}
			},
		},
		{
			name: "create completed milestone",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"title":   "Completed Milestone",
				"status":  StatusCompleted,
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				milestone := result["milestone"].(Milestone)
				if milestone.Status != StatusCompleted {
					t.Errorf("Expected status 'completed'")
				}
				if milestone.CompletedAt.IsZero() {
					t.Errorf("Expected completed_at to be set")
				}
			},
		},
		{
			name: "invalid due_date format",
			params: map[string]interface{}{
				"plan_id":  plan.ID,
				"title":    "Invalid Date",
				"due_date": "invalid-date",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["success"].(bool) {
					t.Errorf("Expected success=false for invalid date")
				}
			},
		},
		{
			name: "invalid status",
			params: map[string]interface{}{
				"plan_id": plan.ID,
				"title":   "Invalid Status",
				"status":  "invalid",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["success"].(bool) {
					t.Errorf("Expected success=false for invalid status")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CreateMilestone(ctx, tt.params)
			if err != nil {
				t.Fatalf("CreateMilestone returned error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestTrackProgress(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	createResult, _ := CreatePlan(ctx, map[string]interface{}{
		"title": "Progress Test Plan",
	})
	plan := createResult["plan"].(Plan)

	CreateMilestone(ctx, map[string]interface{}{
		"plan_id": plan.ID,
		"title":   "Milestone 1",
		"status":  StatusCompleted,
	})
	CreateMilestone(ctx, map[string]interface{}{
		"plan_id": plan.ID,
		"title":   "Milestone 2",
		"status":  StatusInProgress,
	})
	CreateMilestone(ctx, map[string]interface{}{
		"plan_id": plan.ID,
		"title":   "Milestone 3",
		"status":  StatusPending,
	})

	tests := []struct {
		name     string
		params   map[string]interface{}
		validate func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "track progress with milestones",
			params: map[string]interface{}{
				"plan_id": plan.ID,
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				progress := result["progress"].(float64)
				expectedProgress := 100.0 / 3.0
				if progress < expectedProgress-0.1 || progress > expectedProgress+0.1 {
					t.Errorf("Expected progress ~%.1f%%, got %.1f%%", expectedProgress, progress)
				}
				if result["total_milestones"].(int) != 3 {
					t.Errorf("Expected 3 total milestones")
				}
				if result["completed_milestones"].(int) != 1 {
					t.Errorf("Expected 1 completed milestone")
				}
				if result["in_progress_milestones"].(int) != 1 {
					t.Errorf("Expected 1 in_progress milestone")
				}
			},
		},
		{
			name: "non-existent plan",
			params: map[string]interface{}{
				"plan_id": "non-existent",
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if result["success"].(bool) {
					t.Errorf("Expected success=false for non-existent plan")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TrackProgress(ctx, tt.params)
			if err != nil {
				t.Fatalf("TrackProgress returned error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestTrackProgressNoMilestones(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	createResult, _ := CreatePlan(ctx, map[string]interface{}{
		"title": "Plan with No Milestones",
	})
	plan := createResult["plan"].(Plan)

	result, err := TrackProgress(ctx, map[string]interface{}{
		"plan_id": plan.ID,
	})
	if err != nil {
		t.Fatalf("TrackProgress returned error: %v", err)
	}

	if !result["success"].(bool) {
		t.Fatalf("Expected success=true")
	}
	if result["progress"].(float64) != 0.0 {
		t.Errorf("Expected progress 0.0 for no milestones")
	}
}

func TestTrackProgressAllCompleted(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	createResult, _ := CreatePlan(ctx, map[string]interface{}{
		"title": "All Completed Plan",
	})
	plan := createResult["plan"].(Plan)

	CreateMilestone(ctx, map[string]interface{}{
		"plan_id": plan.ID,
		"title":   "Milestone 1",
		"status":  StatusCompleted,
	})
	CreateMilestone(ctx, map[string]interface{}{
		"plan_id": plan.ID,
		"title":   "Milestone 2",
		"status":  StatusCompleted,
	})

	result, err := TrackProgress(ctx, map[string]interface{}{
		"plan_id": plan.ID,
	})
	if err != nil {
		t.Fatalf("TrackProgress returned error: %v", err)
	}

	if !result["success"].(bool) {
		t.Fatalf("Expected success=true")
	}
	if result["progress"].(float64) != 100.0 {
		t.Errorf("Expected progress 100.0, got %.1f", result["progress"].(float64))
	}
	if result["status"].(string) != StatusCompleted {
		t.Errorf("Expected status to be 'completed', got '%s'", result["status"].(string))
	}
}

func TestValidatePlan(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	tests := []struct {
		name        string
		setupPlan   func() string
		validate    func(t *testing.T, result map[string]interface{})
		description string
	}{
		{
			name: "valid plan with all fields",
			setupPlan: func() string {
				result, _ := CreatePlan(ctx, map[string]interface{}{
					"title":       "Valid Plan",
					"description": "A complete plan",
					"goals":       []string{"Goal 1", "Goal 2"},
				})
				plan := result["plan"].(Plan)
				CreateMilestone(ctx, map[string]interface{}{
					"plan_id":     plan.ID,
					"title":       "Milestone 1",
					"description": "First milestone",
				})
				return plan.ID
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				if !result["valid"].(bool) {
					t.Errorf("Expected valid=true")
				}
				issues := result["issues"].([]string)
				if len(issues) != 0 {
					t.Errorf("Expected 0 issues, got %d: %v", len(issues), issues)
				}
			},
		},
		{
			name: "plan with no goals",
			setupPlan: func() string {
				result, _ := CreatePlan(ctx, map[string]interface{}{
					"title": "No Goals Plan",
				})
				return result["plan"].(Plan).ID
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				warnings := result["warnings"].([]string)
				found := false
				for _, w := range warnings {
					if w == "Plan has no goals defined" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected warning about no goals")
				}
			},
		},
		{
			name: "plan with duplicate milestones",
			setupPlan: func() string {
				result, _ := CreatePlan(ctx, map[string]interface{}{
					"title": "Duplicate Milestones Plan",
				})
				plan := result["plan"].(Plan)
				CreateMilestone(ctx, map[string]interface{}{
					"plan_id": plan.ID,
					"title":   "Duplicate",
				})
				CreateMilestone(ctx, map[string]interface{}{
					"plan_id": plan.ID,
					"title":   "Duplicate",
				})
				return plan.ID
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				warnings := result["warnings"].([]string)
				foundDuplicate := false
				for _, w := range warnings {
					if w == "Duplicate milestone title: 'Duplicate' (appears 2 times)" {
						foundDuplicate = true
						break
					}
				}
				if !foundDuplicate {
					t.Errorf("Expected warning about duplicate milestones, got: %v", warnings)
				}
			},
		},
		{
			name: "plan with overdue milestones",
			setupPlan: func() string {
				result, _ := CreatePlan(ctx, map[string]interface{}{
					"title": "Overdue Plan",
				})
				plan := result["plan"].(Plan)
				CreateMilestone(ctx, map[string]interface{}{
					"plan_id":  plan.ID,
					"title":    "Overdue Milestone",
					"due_date": "2020-01-01",
					"status":   StatusPending,
				})
				return plan.ID
			},
			validate: func(t *testing.T, result map[string]interface{}) {
				if !result["success"].(bool) {
					t.Fatalf("Expected success=true")
				}
				warnings := result["warnings"].([]string)
				foundOverdue := false
				for _, w := range warnings {
					if w == "1 milestone(s) are overdue" {
						foundOverdue = true
						break
					}
				}
				if !foundOverdue {
					t.Errorf("Expected warning about overdue milestones, got: %v", warnings)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planID := tt.setupPlan()
			result, err := ValidatePlan(ctx, map[string]interface{}{
				"plan_id": planID,
			})
			if err != nil {
				t.Fatalf("ValidatePlan returned error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestPlanPersistence(t *testing.T) {
	plansDir := setupTestPlansDir(t)
	ctx := context.Background()

	createResult, _ := CreatePlan(ctx, map[string]interface{}{
		"title":       "Persistent Plan",
		"description": "Test persistence",
		"goals":       []string{"Goal 1"},
	})
	plan := createResult["plan"].(Plan)

	planPath := filepath.Join(plansDir, plan.ID+".json")
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		t.Fatalf("Plan file was not created at %s", planPath)
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatalf("Failed to read plan file: %v", err)
	}

	var loadedPlan Plan
	if err := json.Unmarshal(data, &loadedPlan); err != nil {
		t.Fatalf("Failed to unmarshal plan: %v", err)
	}

	if loadedPlan.ID != plan.ID {
		t.Errorf("Expected ID %s, got %s", plan.ID, loadedPlan.ID)
	}
	if loadedPlan.Title != plan.Title {
		t.Errorf("Expected Title %s, got %s", plan.Title, loadedPlan.Title)
	}
}

func TestToolRegistration(t *testing.T) {
	tools := []string{
		"create_plan",
		"update_plan",
		"track_progress",
		"validate_plan",
		"create_milestone",
	}

	for _, toolName := range tools {
		t.Run(toolName, func(t *testing.T) {
			tool, err := GetTool(toolName)
			if err != nil {
				t.Fatalf("Tool %s not registered: %v", toolName, err)
			}
			if tool.Name != toolName {
				t.Errorf("Expected tool name %s, got %s", toolName, tool.Name)
			}
			if tool.Function == nil {
				t.Errorf("Tool %s has nil function", toolName)
			}
			if tool.Description == "" {
				t.Errorf("Tool %s has empty description", toolName)
			}
			if tool.Parameters == nil {
				t.Errorf("Tool %s has nil parameters", toolName)
			}
		})
	}
}
