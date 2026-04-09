package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/DojoGenesis/gateway/channel"
	"github.com/DojoGenesis/gateway/runtime/cas"
	"github.com/DojoGenesis/gateway/workflow"
)

// ---------------------------------------------------------------------------
// memPublisher — in-memory NATSPublisher+NATSSubscriber for integration tests.
// Records all published events and delivers them to subscribers synchronously,
// simulating embedded NATS without external infrastructure.
// ---------------------------------------------------------------------------

type memPublishedEvent struct {
	Subject string
	Data    []byte
}

type memPublisher struct {
	mu       sync.Mutex
	events   []memPublishedEvent
	handlers []func(subject string, data []byte)
	ackCount atomic.Int64
}

func (p *memPublisher) PublishRaw(_ context.Context, subject string, data []byte) error {
	p.mu.Lock()
	p.events = append(p.events, memPublishedEvent{Subject: subject, Data: data})
	handlers := make([]func(string, []byte), len(p.handlers))
	copy(handlers, p.handlers)
	p.mu.Unlock()

	// Deliver to subscribed handlers synchronously (simulates NATS delivery).
	for _, h := range handlers {
		h(subject, data)
	}

	// Count as acknowledged (simulates JetStream ack).
	p.ackCount.Add(1)

	return nil
}

func (p *memPublisher) SubscribeRaw(_ context.Context, _ string, handler func(string, []byte)) (func(), error) {
	p.mu.Lock()
	p.handlers = append(p.handlers, handler)
	idx := len(p.handlers) - 1
	p.mu.Unlock()

	return func() {
		p.mu.Lock()
		if idx < len(p.handlers) {
			p.handlers[idx] = func(string, []byte) {} // no-op
		}
		p.mu.Unlock()
	}, nil
}

func (p *memPublisher) published() []memPublishedEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]memPublishedEvent, len(p.events))
	copy(out, p.events)
	return out
}

func (p *memPublisher) acknowledged() int64 {
	return p.ackCount.Load()
}

// ---------------------------------------------------------------------------
// stubWorkflowRunner — satisfies channel.WorkflowRunner for integration tests.
// ---------------------------------------------------------------------------

type stubWorkflowRunner struct {
	mu       sync.Mutex
	result   *channel.WorkflowRunResult
	err      error
	calls    []string // workflow names executed
	eventFn  func(workflowName, stepID, status string)
}

func (r *stubWorkflowRunner) Execute(_ context.Context, name string) (*channel.WorkflowRunResult, error) {
	r.mu.Lock()
	r.calls = append(r.calls, name)
	r.mu.Unlock()

	if r.err != nil {
		return nil, r.err
	}

	// Emit step events if an event function is registered.
	if r.eventFn != nil {
		r.eventFn(name, "step-1", "queued")
		r.eventFn(name, "step-1", "running")
		r.eventFn(name, "step-1", "completed")
		r.eventFn(name, "step-2", "queued")
		r.eventFn(name, "step-2", "running")
		r.eventFn(name, "step-2", "completed")
	}

	res := &channel.WorkflowRunResult{
		WorkflowName: name,
		Status:       "completed",
		StepCount:    2,
	}
	if r.result != nil {
		cp := *r.result
		cp.WorkflowName = name
		res = &cp
	}
	return res, nil
}

func (r *stubWorkflowRunner) executedWorkflows() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

// ---------------------------------------------------------------------------
// mockInfisicalClient — mock for InfisicalCredentialStore tests.
// ---------------------------------------------------------------------------

type mockInfisicalClient struct {
	mu      sync.RWMutex
	secrets map[string]string
}

func newMockInfisicalClient() *mockInfisicalClient {
	return &mockInfisicalClient{secrets: make(map[string]string)}
}

func (m *mockInfisicalClient) GetSecret(_ context.Context, key, _, _ string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.secrets[key]
	if !ok {
		return "", fmt.Errorf("secret %q not found", key)
	}
	return v, nil
}

func (m *mockInfisicalClient) ListSecrets(_ context.Context, _, _ string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]string, 0, len(m.secrets))
	for k := range m.secrets {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockInfisicalClient) setSecret(key, value string) {
	m.mu.Lock()
	m.secrets[key] = value
	m.mu.Unlock()
}

// ---------------------------------------------------------------------------
// CAS + workflow helpers
// ---------------------------------------------------------------------------

// createTestCAS creates a temporary SQLite CAS store for testing.
func createTestCAS(t *testing.T) cas.Store {
	t.Helper()
	dbPath := t.TempDir() + "/test_cas.db"
	store, err := cas.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("create test CAS: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// storeTestWorkflow creates a minimal workflow definition and stores it in CAS.
// Returns the CAS ref (sha256 hash).
func storeTestWorkflow(t *testing.T, store cas.Store, name string, steps []workflow.Step) cas.Ref {
	t.Helper()
	def := &workflow.WorkflowDefinition{
		Name:         name,
		Version:      "1.0.0",
		ArtifactType: workflow.WorkflowArtifactType,
		Steps:        steps,
	}

	data, err := workflow.Marshal(def)
	if err != nil {
		t.Fatalf("marshal workflow %q: %v", name, err)
	}

	ctx := context.Background()
	ref, err := store.Put(ctx, data, cas.ContentMeta{
		Type:      cas.ContentConfig,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"kind":    "workflow",
			"name":    name,
			"version": "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("put workflow %q: %v", name, err)
	}

	// Tag as workflow/{name}:latest
	if err := store.Tag(ctx, "workflow/"+name, "latest", ref); err != nil {
		t.Fatalf("tag workflow %q: %v", name, err)
	}
	if err := store.Tag(ctx, "workflow/"+name, "1.0.0", ref); err != nil {
		t.Fatalf("tag workflow %q version: %v", name, err)
	}

	return ref
}

// buildMinimalWorkflowJSON builds a 2-step workflow definition as JSON bytes.
func buildMinimalWorkflowJSON(name string) ([]byte, error) {
	def := workflow.WorkflowDefinition{
		Name:         name,
		Version:      "1.0.0",
		ArtifactType: workflow.WorkflowArtifactType,
		Steps: []workflow.Step{
			{
				ID:    "step-1",
				Skill: "strategic-scout",
				Inputs: map[string]string{
					"topic": "integration test",
				},
			},
			{
				ID:        "step-2",
				Skill:     "research-synthesis",
				DependsOn: []string{"step-1"},
				Inputs: map[string]string{
					"sources": "{{ steps.step-1.outputs.report }}",
				},
			},
		},
	}
	return json.MarshalIndent(def, "", "  ")
}
