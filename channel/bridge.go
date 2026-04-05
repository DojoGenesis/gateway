package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkflowRunner executes a named workflow and returns a minimal result.
// The channel package defines this interface to avoid importing the workflow
// package directly, preserving Phase 0 dependency isolation (ADR-018).
// Use a thin adapter in cmd/dojo (or integration tests) to wrap a real
// workflow.WorkflowExecutor and satisfy this interface.
type WorkflowRunner interface {
	Execute(ctx context.Context, name string) (*WorkflowRunResult, error)
}

// WorkflowRunResult is the bridge-facing view of a workflow execution outcome.
// It contains only what the bridge needs to compose a channel reply.
type WorkflowRunResult struct {
	// WorkflowName is the workflow that ran.
	WorkflowName string

	// Status is the overall outcome: "completed", "failed", "cancelled".
	Status string

	// StepCount is the total number of steps in the workflow.
	StepCount int

	// FailedSteps lists step IDs that did not complete successfully.
	FailedSteps []string
}

// TriggerSpec binds a message pattern to a workflow name.
// When the bridge receives a ChannelMessage matching Platform and Pattern,
// it dispatches to the named Workflow.
type TriggerSpec struct {
	// Platform is the messaging platform to match (e.g. "slack", "discord").
	// Use "*" to match any platform.
	Platform string

	// Pattern is a compiled regular expression matched against the message
	// text. A nil Pattern matches all messages on the given platform.
	Pattern *regexp.Regexp

	// Workflow is the CAS workflow name to execute on a match.
	Workflow string
}

// ChannelBridge wires channel adapters, a WorkflowRunner, and trigger specs
// into the flywheel:
//
//	HTTP webhook → WebhookGateway → bus.Publish(Event)
//	          → ChannelBridge.HandleEvent
//	          → match TriggerSpec → runner.Execute
//	          → adapter.Send(reply)
//
// Construct with NewChannelBridge. Register adapters and add triggers before
// subscribing to the event bus with BusHandler.
type ChannelBridge struct {
	mu       sync.RWMutex
	adapters map[string]ChannelAdapter
	runner   WorkflowRunner
	triggers []TriggerSpec
}

// NewChannelBridge creates a bridge backed by the given WorkflowRunner.
// runner may be nil; events still match triggers and are logged, but no
// workflow executes and no reply is sent.
func NewChannelBridge(runner WorkflowRunner) *ChannelBridge {
	return &ChannelBridge{
		adapters: make(map[string]ChannelAdapter),
		runner:   runner,
	}
}

// Register adds a ChannelAdapter so the bridge can send replies to the given
// platform. Replaces any existing adapter for the same platform name.
func (b *ChannelBridge) Register(platform string, adapter ChannelAdapter) {
	b.mu.Lock()
	b.adapters[platform] = adapter
	b.mu.Unlock()
	slog.Info("bridge: registered adapter", "platform", platform)
}

// AddTrigger appends a TriggerSpec. Triggers are evaluated in registration
// order; the first match wins.
func (b *ChannelBridge) AddTrigger(spec TriggerSpec) {
	b.mu.Lock()
	b.triggers = append(b.triggers, spec)
	b.mu.Unlock()
	slog.Info("bridge: registered trigger",
		"platform", spec.Platform,
		"workflow", spec.Workflow,
	)
}

// BusHandler returns a subscriber function for InProcessBus.Subscribe.
// Typical wiring:
//
//	bus    := &channel.InProcessBus{}
//	gw     := channel.NewWebhookGateway(bus, creds)
//	bridge := channel.NewChannelBridge(runner)
//	gw.Register("slack", slackAdapter)
//	bridge.Register("slack", slackAdapter)
//	bridge.AddTrigger(channel.TriggerSpec{Platform: "slack", Workflow: "my-flow"})
//	bus.Subscribe(bridge.BusHandler(ctx))
func (b *ChannelBridge) BusHandler(ctx context.Context) func(string, Event) {
	return func(_ string, evt Event) {
		if err := b.HandleEvent(ctx, evt); err != nil {
			slog.Error("bridge: handle event error", "error", err)
		}
	}
}

// HandleEvent processes a CloudEvent emitted by WebhookGateway.
//
//  1. Unmarshals the event data into a ChannelMessage.
//  2. Finds the first matching TriggerSpec.
//  3. Executes the workflow via runner.Execute.
//  4. Sends a reply back via the platform's registered adapter.
//
// Structural failures (bad JSON) return an error. Workflow execution and
// send errors are logged but do not propagate — the bridge always attempts
// to complete the round trip.
func (b *ChannelBridge) HandleEvent(ctx context.Context, evt Event) error {
	var msg ChannelMessage
	if err := json.Unmarshal(evt.Data, &msg); err != nil {
		return fmt.Errorf("bridge: unmarshal event data: %w", err)
	}

	b.mu.RLock()
	spec, matched := b.matchTrigger(msg)
	b.mu.RUnlock()

	if !matched {
		slog.Debug("bridge: no trigger matched",
			"platform", msg.Platform,
			"text", msg.Text,
		)
		return nil
	}

	slog.Info("bridge: trigger matched",
		"platform", msg.Platform,
		"workflow", spec.Workflow,
	)

	if b.runner == nil {
		slog.Warn("bridge: trigger matched but runner is nil — no workflow executed",
			"workflow", spec.Workflow,
		)
		return nil
	}

	result, execErr := b.runner.Execute(ctx, spec.Workflow)
	reply := b.buildReply(&msg, spec.Workflow, result, execErr)

	b.mu.RLock()
	adapter, ok := b.adapters[msg.Platform]
	b.mu.RUnlock()

	if !ok {
		slog.Warn("bridge: no adapter for platform, reply dropped",
			"platform", msg.Platform,
		)
		return nil
	}

	if err := adapter.Send(ctx, reply); err != nil {
		slog.Error("bridge: send reply failed",
			"platform", msg.Platform,
			"error", err,
		)
	}

	return nil
}

// matchTrigger returns the first TriggerSpec matching msg.
// Must be called with b.mu.RLock held.
func (b *ChannelBridge) matchTrigger(msg ChannelMessage) (TriggerSpec, bool) {
	for _, spec := range b.triggers {
		if spec.Platform != "*" && spec.Platform != msg.Platform {
			continue
		}
		if spec.Pattern != nil && !spec.Pattern.MatchString(msg.Text) {
			continue
		}
		return spec, true
	}
	return TriggerSpec{}, false
}

// buildReply constructs a reply ChannelMessage directed at the original
// channel and thread.
func (b *ChannelBridge) buildReply(
	original *ChannelMessage,
	workflowName string,
	result *WorkflowRunResult,
	execErr error,
) *ChannelMessage {
	var text string
	switch {
	case execErr != nil:
		text = fmt.Sprintf("Workflow %q failed to start: %s", workflowName, execErr)
	case result == nil:
		text = fmt.Sprintf("Workflow %q dispatched.", workflowName)
	case result.Status == "completed":
		text = fmt.Sprintf("Workflow %q completed (%d steps).", workflowName, result.StepCount)
	case result.Status == "failed":
		text = fmt.Sprintf("Workflow %q finished with failures: %v", workflowName, result.FailedSteps)
	default:
		text = fmt.Sprintf("Workflow %q status: %s", workflowName, result.Status)
	}

	return &ChannelMessage{
		ID:        uuid.New().String(),
		Platform:  original.Platform,
		ChannelID: original.ChannelID,
		UserID:    "dojo-bot",
		UserName:  "Dojo",
		Text:      text,
		Timestamp: time.Now().UTC(),
		ReplyTo:   original.ID,
		ThreadID:  original.ThreadID,
	}
}

// ---------------------------------------------------------------------------
// InProcessBus — synchronous in-process event bus.
// For production replace with the NATS-backed bus from runtime/event.
// ---------------------------------------------------------------------------

// InProcessBus is a synchronous in-process event bus. It implements
// EventPublisher (passed to WebhookGateway) and exposes Subscribe so
// ChannelBridge can register a handler via BusHandler.
//
// Subscribers are called synchronously in the Publish goroutine, which
// guarantees ordering in tests and keeps the flywheel deterministic during
// early development. Production deployments should use the NATS bus.
type InProcessBus struct {
	mu          sync.RWMutex
	subscribers []func(string, Event)
}

// Publish calls all registered subscribers with subject and event.
// Implements EventPublisher.
func (b *InProcessBus) Publish(subject string, evt Event) error {
	b.mu.RLock()
	subs := make([]func(string, Event), len(b.subscribers))
	copy(subs, b.subscribers)
	b.mu.RUnlock()

	for _, sub := range subs {
		sub(subject, evt)
	}
	return nil
}

// Subscribe registers fn to be called for every published event.
func (b *InProcessBus) Subscribe(fn func(subject string, evt Event)) {
	b.mu.Lock()
	b.subscribers = append(b.subscribers, fn)
	b.mu.Unlock()
}
