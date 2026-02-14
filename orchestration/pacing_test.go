package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition"
)

func TestPacingDelay_Deliberate(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing: "deliberate",
	}

	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
		WithDisposition(disp),
	)

	delay := engine.pacingDelay()
	expected := 3 * time.Second

	if delay != expected {
		t.Errorf("deliberate pacing: expected %v, got %v", expected, delay)
	}
}

func TestPacingDelay_Measured(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing: "measured",
	}

	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
		WithDisposition(disp),
	)

	delay := engine.pacingDelay()
	expected := 1500 * time.Millisecond

	if delay != expected {
		t.Errorf("measured pacing: expected %v, got %v", expected, delay)
	}
}

func TestPacingDelay_Responsive(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing: "responsive",
	}

	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
		WithDisposition(disp),
	)

	delay := engine.pacingDelay()
	expected := 750 * time.Millisecond

	if delay != expected {
		t.Errorf("responsive pacing: expected %v, got %v", expected, delay)
	}
}

func TestPacingDelay_Rapid(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing: "rapid",
	}

	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
		WithDisposition(disp),
	)

	delay := engine.pacingDelay()
	expected := time.Duration(0)

	if delay != expected {
		t.Errorf("rapid pacing: expected %v, got %v", expected, delay)
	}
}

func TestApplyPacingDelay_Timing(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing: "responsive",
	}

	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
		WithDisposition(disp),
	)

	ctx := context.Background()
	start := time.Now()

	err := engine.applyPacingDelay(ctx)
	if err != nil {
		t.Fatalf("applyPacingDelay failed: %v", err)
	}

	elapsed := time.Since(start)
	expectedMin := 700 * time.Millisecond
	expectedMax := 800 * time.Millisecond

	if elapsed < expectedMin || elapsed > expectedMax {
		t.Errorf("pacing delay timing: expected %v-%v, got %v", expectedMin, expectedMax, elapsed)
	}
}

func TestApplyPacingDelay_CanceledContext(t *testing.T) {
	disp := &disposition.DispositionConfig{
		Pacing: "deliberate",
	}

	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
		WithDisposition(disp),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := engine.applyPacingDelay(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestPacingDelay_DefaultDisposition(t *testing.T) {
	// Engine without explicit disposition should use DefaultDisposition
	engine := NewEngine(
		DefaultEngineConfig(),
		nil, nil, nil, nil, nil,
	)

	delay := engine.pacingDelay()
	// DefaultDisposition() uses "measured" pacing
	expected := 1500 * time.Millisecond

	if delay != expected {
		t.Errorf("default disposition pacing: expected %v, got %v", expected, delay)
	}
}
