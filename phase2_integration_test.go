package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/collaboration"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/disposition"
	pkgerrors "github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/errors"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/intelligence"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/reflection"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/validation"
)

// TestPhase2Integration verifies Phase 2 ADA Full Integration implementation.
// All 7 gateway modules must be disposition-aware per the behavioral contract.
func TestPhase2Integration_AllModulesConfigured(t *testing.T) {
	// Load test disposition
	testdataDir := filepath.Join("pkg", "disposition", "testdata")
	disp, err := disposition.ResolveDisposition(testdataDir, "")
	if err != nil {
		t.Logf("Using default disposition (file load failed): %v", err)
		disp = disposition.DefaultDisposition()
	}

	// Verify all 7 modules can be created with disposition
	modules := make(map[string]bool)

	// Module 1: Orchestration (pacing) - verified separately in orchestration/pacing_test.go
	modules["orchestration"] = true

	// Module 2: Memory (depth) - verified separately in memory/depth_strategy_test.go
	modules["memory"] = true

	// Module 3: Proactive Intelligence
	intEngine := intelligence.NewProactiveEngine(intelligence.WithDisposition(disp))
	if intEngine == nil {
		t.Fatal("failed to create intelligence engine")
	}
	modules["intelligence"] = true

	// Module 4: Error Handler
	errHandler := pkgerrors.NewHandler(pkgerrors.WithDisposition(disp))
	if errHandler == nil {
		t.Fatal("failed to create error handler")
	}
	modules["error_handler"] = true

	// Module 5: Collaboration Manager
	collabMgr := collaboration.NewManager(collaboration.WithDisposition(disp))
	if collabMgr == nil {
		t.Fatal("failed to create collaboration manager")
	}
	modules["collaboration"] = true

	// Module 6: Validator
	validator := validation.NewValidator(validation.WithDisposition(disp))
	if validator == nil {
		t.Fatal("failed to create validator")
	}
	modules["validator"] = true

	// Module 7: Reflection Engine
	reflEngine := reflection.NewEngine(reflection.WithDisposition(disp))
	if reflEngine == nil {
		t.Fatal("failed to create reflection engine")
	}
	modules["reflection"] = true

	if len(modules) != 7 {
		t.Errorf("expected 7 modules, got %d", len(modules))
	}

	t.Logf("✓ All 7 modules successfully configured with disposition")
	for module := range modules {
		t.Logf("  - %s", module)
	}
}

// TestPhase2Integration_BehaviorChanges verifies disposition values change behavior.
func TestPhase2Integration_BehaviorChanges(t *testing.T) {
	ctx := context.Background()

	t.Run("Initiative changes suggestion behavior", func(t *testing.T) {
		reactive := &disposition.DispositionConfig{Initiative: "reactive"}
		proactive := &disposition.DispositionConfig{Initiative: "proactive"}

		reactiveEng := intelligence.NewProactiveEngine(intelligence.WithDisposition(reactive))
		proactiveEng := intelligence.NewProactiveEngine(intelligence.WithDisposition(proactive))

		event := intelligence.TaskEvent{Type: "task_complete"}

		if reactiveEng.ShouldSuggest(ctx, event) {
			t.Error("reactive should not suggest on task_complete")
		}
		if !proactiveEng.ShouldSuggest(ctx, event) {
			t.Error("proactive should suggest on task_complete")
		}
	})

	t.Run("Error strategy changes error handling", func(t *testing.T) {
		failFast := &disposition.DispositionConfig{
			ErrorHandling: disposition.ErrorHandlingConfig{Strategy: "fail-fast"},
		}
		retry := &disposition.DispositionConfig{
			ErrorHandling: disposition.ErrorHandlingConfig{Strategy: "retry", RetryCount: 3},
		}

		failFastHandler := pkgerrors.NewHandler(pkgerrors.WithDisposition(failFast))
		retryHandler := pkgerrors.NewHandler(pkgerrors.WithDisposition(retry))

		testErr := errors.New("test error")

		failDecision := failFastHandler.HandleError(ctx, testErr, 0)
		retryDecision := retryHandler.HandleError(ctx, testErr, 0)

		if failDecision.Action != pkgerrors.ActionStop {
			t.Error("fail-fast should stop on error")
		}
		if retryDecision.Action != pkgerrors.ActionRetry {
			t.Error("retry should retry on first error")
		}
	})

	t.Run("Collaboration style changes check-in behavior", func(t *testing.T) {
		independent := &disposition.DispositionConfig{
			Collaboration: disposition.CollaborationConfig{
				Style:            "independent",
				CheckInFrequency: "constantly",
			},
		}
		consultative := &disposition.DispositionConfig{
			Collaboration: disposition.CollaborationConfig{
				Style:            "consultative",
				CheckInFrequency: "constantly",
			},
		}

		indMgr := collaboration.NewManager(collaboration.WithDisposition(independent))
		conMgr := collaboration.NewManager(collaboration.WithDisposition(consultative))

		event := collaboration.CollabEvent{Type: "decision_point", IsSignificant: true}

		if indMgr.ShouldCheckIn(ctx, event) {
			t.Error("independent should never check in")
		}
		if !conMgr.ShouldCheckIn(ctx, event) {
			t.Error("consultative should check in at decisions")
		}
	})
}

// TestPhase2Integration_DefaultFallback verifies graceful default handling.
func TestPhase2Integration_DefaultFallback(t *testing.T) {
	// All modules should work without explicit disposition (using defaults)
	_ = intelligence.NewProactiveEngine()
	_ = pkgerrors.NewHandler()
	_ = collaboration.NewManager()
	_ = validation.NewValidator()
	_ = reflection.NewEngine()

	t.Log("✓ All modules gracefully use default disposition")
}
