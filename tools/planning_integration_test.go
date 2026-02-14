package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanningIntegrationWorkflow(t *testing.T) {
	plansDir := setupTestPlansDir(t)
	ctx := context.Background()

	t.Run("complete workflow: create plan, add milestones, track progress", func(t *testing.T) {
		createResult, err := CreatePlan(ctx, map[string]interface{}{
			"title":       "Product Launch Plan",
			"description": "Launch new product to market",
			"goals": []string{
				"Complete development",
				"Beta testing",
				"Marketing campaign",
				"Product launch",
			},
			"metadata": map[string]interface{}{
				"owner":    "Product Team",
				"priority": "high",
				"quarter":  "Q1 2024",
			},
		})
		if err != nil {
			t.Fatalf("CreatePlan failed: %v", err)
		}

		if !createResult["success"].(bool) {
			t.Fatalf("CreatePlan returned success=false: %v", createResult["error"])
		}

		plan := createResult["plan"].(Plan)
		planID := plan.ID

		milestoneResults := []map[string]interface{}{}

		milestone1, _ := CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     planID,
			"title":       "Complete Alpha Version",
			"description": "Finish core features for alpha release",
			"status":      StatusCompleted,
			"due_date":    "2024-01-31",
		})
		milestoneResults = append(milestoneResults, milestone1)

		milestone2, _ := CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     planID,
			"title":       "Beta Testing",
			"description": "Run beta tests with select users",
			"status":      StatusInProgress,
			"due_date":    "2024-02-28",
		})
		milestoneResults = append(milestoneResults, milestone2)

		milestone3, _ := CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     planID,
			"title":       "Marketing Campaign",
			"description": "Launch marketing materials",
			"status":      StatusPending,
			"due_date":    "2024-03-15",
		})
		milestoneResults = append(milestoneResults, milestone3)

		milestone4, _ := CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     planID,
			"title":       "Product Launch",
			"description": "Official product launch event",
			"status":      StatusPending,
			"due_date":    "2024-03-31",
		})
		milestoneResults = append(milestoneResults, milestone4)

		for i, result := range milestoneResults {
			if !result["success"].(bool) {
				t.Errorf("CreateMilestone %d failed: %v", i+1, result["error"])
			}
		}

		progressResult, err := TrackProgress(ctx, map[string]interface{}{
			"plan_id": planID,
		})
		if err != nil {
			t.Fatalf("TrackProgress failed: %v", err)
		}

		if !progressResult["success"].(bool) {
			t.Fatalf("TrackProgress returned success=false")
		}

		progress := progressResult["progress"].(float64)
		expectedProgress := 25.0
		if progress != expectedProgress {
			t.Errorf("Expected progress %.1f%%, got %.1f%%", expectedProgress, progress)
		}

		if progressResult["completed_milestones"].(int) != 1 {
			t.Errorf("Expected 1 completed milestone, got %d", progressResult["completed_milestones"].(int))
		}

		if progressResult["in_progress_milestones"].(int) != 1 {
			t.Errorf("Expected 1 in_progress milestone, got %d", progressResult["in_progress_milestones"].(int))
		}

		validateResult, err := ValidatePlan(ctx, map[string]interface{}{
			"plan_id": planID,
		})
		if err != nil {
			t.Fatalf("ValidatePlan failed: %v", err)
		}

		if !validateResult["success"].(bool) {
			t.Fatalf("ValidatePlan returned success=false")
		}

		if !validateResult["valid"].(bool) {
			t.Errorf("Expected plan to be valid, got issues: %v", validateResult["issues"])
		}

		updateResult, err := UpdatePlan(ctx, map[string]interface{}{
			"plan_id":     planID,
			"description": "Updated: Launch new product to market with extensive beta testing",
			"metadata": map[string]interface{}{
				"last_updated": "2024-01-15",
			},
		})
		if err != nil {
			t.Fatalf("UpdatePlan failed: %v", err)
		}

		if !updateResult["success"].(bool) {
			t.Fatalf("UpdatePlan returned success=false")
		}

		updatedPlan := updateResult["plan"].(*Plan)
		if updatedPlan.Metadata["last_updated"] != "2024-01-15" {
			t.Errorf("Expected metadata to be updated")
		}

		planPath := filepath.Join(plansDir, planID+".json")
		data, err := os.ReadFile(planPath)
		if err != nil {
			t.Fatalf("Failed to read plan file: %v", err)
		}

		var savedPlan Plan
		if err := json.Unmarshal(data, &savedPlan); err != nil {
			t.Fatalf("Failed to unmarshal saved plan: %v", err)
		}

		if len(savedPlan.Milestones) != 4 {
			t.Errorf("Expected 4 milestones in saved plan, got %d", len(savedPlan.Milestones))
		}

		if savedPlan.Progress != 25.0 {
			t.Errorf("Expected saved progress 25.0%%, got %.1f%%", savedPlan.Progress)
		}
	})

	t.Run("plan with all milestones completed should be marked completed", func(t *testing.T) {
		createResult, _ := CreatePlan(ctx, map[string]interface{}{
			"title": "Quick Project",
		})
		plan := createResult["plan"].(Plan)
		planID := plan.ID

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id": planID,
			"title":   "Task 1",
			"status":  StatusCompleted,
		})

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id": planID,
			"title":   "Task 2",
			"status":  StatusCompleted,
		})

		progressResult, _ := TrackProgress(ctx, map[string]interface{}{
			"plan_id": planID,
		})

		if progressResult["progress"].(float64) != 100.0 {
			t.Errorf("Expected progress 100.0%%")
		}

		if progressResult["status"].(string) != StatusCompleted {
			t.Errorf("Expected plan status to be 'completed', got '%s'", progressResult["status"].(string))
		}
	})

	t.Run("multiple plans can coexist", func(t *testing.T) {
		plan1Result, _ := CreatePlan(ctx, map[string]interface{}{
			"title": "Plan 1",
		})
		plan1 := plan1Result["plan"].(Plan)

		plan2Result, _ := CreatePlan(ctx, map[string]interface{}{
			"title": "Plan 2",
		})
		plan2 := plan2Result["plan"].(Plan)

		plan3Result, _ := CreatePlan(ctx, map[string]interface{}{
			"title": "Plan 3",
		})
		plan3 := plan3Result["plan"].(Plan)

		files, err := os.ReadDir(plansDir)
		if err != nil {
			t.Fatalf("Failed to read plans directory: %v", err)
		}

		jsonFiles := 0
		for _, f := range files {
			if filepath.Ext(f.Name()) == ".json" {
				jsonFiles++
			}
		}

		if jsonFiles < 3 {
			t.Errorf("Expected at least 3 plan files, found %d", jsonFiles)
		}

		loadedPlan1, err := loadPlan(plan1.ID)
		if err != nil {
			t.Fatalf("Failed to load plan1: %v", err)
		}
		if loadedPlan1.Title != "Plan 1" {
			t.Errorf("Expected plan1 title 'Plan 1', got '%s'", loadedPlan1.Title)
		}

		loadedPlan2, err := loadPlan(plan2.ID)
		if err != nil {
			t.Fatalf("Failed to load plan2: %v", err)
		}
		if loadedPlan2.Title != "Plan 2" {
			t.Errorf("Expected plan2 title 'Plan 2', got '%s'", loadedPlan2.Title)
		}

		loadedPlan3, err := loadPlan(plan3.ID)
		if err != nil {
			t.Fatalf("Failed to load plan3: %v", err)
		}
		if loadedPlan3.Title != "Plan 3" {
			t.Errorf("Expected plan3 title 'Plan 3', got '%s'", loadedPlan3.Title)
		}
	})

	t.Run("plan validation catches common issues", func(t *testing.T) {
		result, _ := CreatePlan(ctx, map[string]interface{}{
			"title": "Plan with Issues",
		})
		plan := result["plan"].(Plan)
		planID := plan.ID

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id":  planID,
			"title":    "Duplicate",
			"due_date": "2020-01-01",
			"status":   StatusPending,
		})

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id":  planID,
			"title":    "Duplicate",
			"due_date": "2020-01-02",
			"status":   StatusPending,
		})

		validateResult, _ := ValidatePlan(ctx, map[string]interface{}{
			"plan_id": planID,
		})

		warnings := validateResult["warnings"].([]string)
		if len(warnings) == 0 {
			t.Errorf("Expected warnings about duplicate milestones and overdue dates")
		}

		foundDuplicate := false
		foundOverdue := false
		for _, w := range warnings {
			if w == "Duplicate milestone title: 'Duplicate' (appears 2 times)" {
				foundDuplicate = true
			}
			if w == "2 milestone(s) are overdue" {
				foundOverdue = true
			}
		}

		if !foundDuplicate {
			t.Errorf("Expected warning about duplicate milestones")
		}
		if !foundOverdue {
			t.Errorf("Expected warning about overdue milestones")
		}
	})
}

func TestPlanningRealWorldScenarios(t *testing.T) {
	setupTestPlansDir(t)
	ctx := context.Background()

	t.Run("software development sprint planning", func(t *testing.T) {
		createResult, _ := CreatePlan(ctx, map[string]interface{}{
			"title":       "Sprint 42 - User Authentication",
			"description": "Implement user authentication system with OAuth2",
			"goals": []string{
				"Implement OAuth2 provider integration",
				"Create user registration flow",
				"Implement password reset",
				"Add 2FA support",
			},
			"metadata": map[string]interface{}{
				"sprint":       42,
				"team":         "Backend Team",
				"start_date":   "2024-02-01",
				"end_date":     "2024-02-14",
				"story_points": 34,
			},
		})

		plan := createResult["plan"].(Plan)

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     plan.ID,
			"title":       "OAuth2 Integration",
			"description": "Integrate Google and GitHub OAuth2 providers",
			"status":      StatusCompleted,
			"due_date":    "2024-02-05",
			"metadata": map[string]interface{}{
				"story_points": 8,
				"assignee":     "alice",
			},
		})

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     plan.ID,
			"title":       "User Registration",
			"description": "Build email/password registration flow",
			"status":      StatusInProgress,
			"due_date":    "2024-02-08",
			"metadata": map[string]interface{}{
				"story_points": 5,
				"assignee":     "bob",
			},
		})

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     plan.ID,
			"title":       "Password Reset",
			"description": "Implement forgot password flow",
			"status":      StatusPending,
			"due_date":    "2024-02-10",
			"metadata": map[string]interface{}{
				"story_points": 5,
				"assignee":     "charlie",
			},
		})

		CreateMilestone(ctx, map[string]interface{}{
			"plan_id":     plan.ID,
			"title":       "2FA Support",
			"description": "Add TOTP-based 2FA",
			"status":      StatusPending,
			"due_date":    "2024-02-14",
			"metadata": map[string]interface{}{
				"story_points": 13,
				"assignee":     "dave",
			},
		})

		progressResult, _ := TrackProgress(ctx, map[string]interface{}{
			"plan_id": plan.ID,
		})

		if !progressResult["success"].(bool) {
			t.Fatalf("Expected successful progress tracking")
		}

		if progressResult["completed_milestones"].(int) != 1 {
			t.Errorf("Expected 1 completed milestone")
		}

		validateResult, _ := ValidatePlan(ctx, map[string]interface{}{
			"plan_id": plan.ID,
		})

		if !validateResult["valid"].(bool) {
			t.Errorf("Expected valid plan, got issues: %v", validateResult["issues"])
		}
	})

	t.Run("marketing campaign planning", func(t *testing.T) {
		createResult, _ := CreatePlan(ctx, map[string]interface{}{
			"title":       "Q1 2024 Marketing Campaign",
			"description": "Launch comprehensive marketing campaign for new product",
			"goals": []string{
				"Increase brand awareness by 50%",
				"Generate 10,000 qualified leads",
				"Achieve 15% conversion rate",
			},
			"metadata": map[string]interface{}{
				"budget":     "$50,000",
				"department": "Marketing",
				"kpis": map[string]interface{}{
					"awareness_target":  "50%",
					"leads_target":      10000,
					"conversion_target": "15%",
				},
			},
		})

		plan := createResult["plan"].(Plan)

		milestones := []map[string]interface{}{
			{
				"title":       "Content Creation",
				"description": "Create blog posts, videos, and social media content",
				"status":      StatusCompleted,
				"due_date":    "2024-01-15",
			},
			{
				"title":       "Email Campaign",
				"description": "Design and launch email drip campaign",
				"status":      StatusCompleted,
				"due_date":    "2024-01-20",
			},
			{
				"title":       "Social Media Ads",
				"description": "Launch targeted social media advertising",
				"status":      StatusInProgress,
				"due_date":    "2024-02-01",
			},
			{
				"title":       "Influencer Partnerships",
				"description": "Partner with key industry influencers",
				"status":      StatusPending,
				"due_date":    "2024-02-15",
			},
			{
				"title":       "Campaign Analysis",
				"description": "Analyze campaign performance and ROI",
				"status":      StatusPending,
				"due_date":    "2024-03-31",
			},
		}

		for _, m := range milestones {
			m["plan_id"] = plan.ID
			result, _ := CreateMilestone(ctx, m)
			if !result["success"].(bool) {
				t.Errorf("Failed to create milestone: %v", result["error"])
			}
		}

		progressResult, _ := TrackProgress(ctx, map[string]interface{}{
			"plan_id": plan.ID,
		})

		progress := progressResult["progress"].(float64)
		expectedProgress := 40.0
		if progress != expectedProgress {
			t.Errorf("Expected %.1f%% progress, got %.1f%%", expectedProgress, progress)
		}
	})
}
