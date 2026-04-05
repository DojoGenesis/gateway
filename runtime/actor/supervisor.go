package actor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("dojo.runtime.actor")

// Metrics instruments registered once at init.
var (
	activeGauge    metric.Int64UpDownCounter
	restartCounter metric.Int64Counter
)

func init() {
	meter := otel.Meter("dojo.runtime.actor")
	var err error
	activeGauge, err = meter.Int64UpDownCounter("dojo.actor.active",
		metric.WithDescription("Number of active agent actors"),
	)
	if err != nil {
		slog.Error("actor: failed to create active gauge", "error", err)
	}
	restartCounter, err = meter.Int64Counter("dojo.actor.restarts",
		metric.WithDescription("Total agent actor restarts"),
	)
	if err != nil {
		slog.Error("actor: failed to create restart counter", "error", err)
	}
}

// messageTypeName returns a human-readable name for a MessageType.
func messageTypeName(t MessageType) string {
	switch t {
	case MessageRequest:
		return "request"
	case MessageResponse:
		return "response"
	case MessageEvent:
		return "event"
	case MessageControl:
		return "control"
	default:
		return "unknown"
	}
}

// Supervisor manages agent actor lifecycle.
type Supervisor interface {
	// Spawn creates a new agent actor with identity and capabilities.
	Spawn(ctx context.Context, opts SpawnOptions) (*AgentRef, error)

	// Send delivers a message to an agent's mailbox (non-blocking).
	Send(ctx context.Context, ref *AgentRef, msg Message) error

	// Request sends a message and waits for a reply.
	Request(ctx context.Context, ref *AgentRef, msg Message, timeout time.Duration) (Message, error)

	// SetStrategy changes the supervision strategy for an agent.
	SetStrategy(ref *AgentRef, strategy SupervisionStrategy) error

	// Shutdown gracefully stops an agent.
	Shutdown(ctx context.Context, ref *AgentRef) error

	// List returns all active agents.
	List(ctx context.Context) ([]*AgentRef, error)

	// Stats returns supervisor metrics.
	Stats() SupervisorStats

	// Close shuts down the supervisor and all agents.
	Close() error
}

// agentSupervisor manages agent actor lifecycle using goroutines and channels.
type agentSupervisor struct {
	mu     sync.RWMutex
	agents map[string]*agentEntry
	config Config
	closed bool

	totalSpawned   atomic.Int64
	totalRestarted atomic.Int64
	totalFailed    atomic.Int64
	msgsHandled    atomic.Int64
}

type agentEntry struct {
	ref        *AgentRef
	handler    AgentHandler
	strategy   SupervisionStrategy
	mailbox    chan Message
	ctx        context.Context
	cancel     context.CancelFunc
	restarts   []time.Time
	done       chan struct{} // signals goroutine exit (#10)
	restarting atomic.Bool  // marks agent as restarting (#4)
}

// NewSupervisor creates a new Supervisor with the given configuration.
func NewSupervisor(config Config) (Supervisor, error) {
	return &agentSupervisor{
		agents: make(map[string]*agentEntry),
		config: config,
	}, nil
}

func (s *agentSupervisor) Spawn(ctx context.Context, opts SpawnOptions) (*AgentRef, error) {
	ctx, span := tracer.Start(ctx, "actor.spawn",
		trace.WithAttributes(
			attribute.String("agent.id", opts.ID),
			attribute.String("identity", opts.IdentityName),
		),
	)
	defer span.End()

	if opts.ID == "" {
		span.RecordError(fmt.Errorf("actor: agent ID is required"))
		return nil, fmt.Errorf("actor: agent ID is required")
	}
	if opts.Handler == nil {
		span.RecordError(fmt.Errorf("actor: handler is required"))
		return nil, fmt.Errorf("actor: handler is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("actor: supervisor is closed")
	}
	if _, exists := s.agents[opts.ID]; exists {
		return nil, fmt.Errorf("actor: agent %q already exists", opts.ID)
	}

	mailboxSize := opts.MailboxSize
	if mailboxSize <= 0 {
		mailboxSize = s.config.DefaultMailboxSize
	}

	strategy := opts.Strategy
	if strategy.MaxRestarts == 0 {
		strategy = s.config.DefaultStrategy
	}

	ref := &AgentRef{
		ID:           opts.ID,
		IdentityName: opts.IdentityName,
	}

	agentCtx, cancel := context.WithCancel(ctx)
	entry := &agentEntry{
		ref:      ref,
		handler:  opts.Handler,
		strategy: strategy,
		mailbox:  make(chan Message, mailboxSize),
		ctx:      agentCtx,
		cancel:   cancel,
		done:     make(chan struct{}),
	}
	s.agents[opts.ID] = entry
	s.totalSpawned.Add(1)
	if activeGauge != nil {
		activeGauge.Add(ctx, 1, metric.WithAttributes(attribute.String("agent.id", opts.ID)))
	}

	// Start the agent goroutine.
	go s.runAgent(entry)

	return ref, nil
}

func (s *agentSupervisor) runAgent(entry *agentEntry) {
	// Capture done channel for THIS goroutine instance. handleFailure may
	// reassign entry.done for the next restart goroutine before our deferred
	// close runs (defers execute LIFO: close-done runs AFTER recover returns).
	myDone := entry.done

	defer func() {
		// Signal that goroutine has exited. (#10)
		select {
		case <-myDone:
			// Already closed (e.g. by a prior restart cycle).
		default:
			close(myDone)
		}
	}()

	// 1.1: Panic recovery — catch panics in message handling and treat as failures.
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("actor: agent %q panicked: %v", entry.ref.ID, r)
			slog.Error("actor: agent panicked", "agent", entry.ref.ID, "panic", r)
			s.handleFailure(entry, err)
		}
	}()

	entry.restarting.Store(false)

	// Call OnStart with a non-cancelled context for the stop handler. (#24)
	if err := entry.handler.OnStart(entry.ctx, entry.ref); err != nil {
		s.handleFailure(entry, err)
		return
	}

	// Process messages.
	for {
		select {
		case <-entry.ctx.Done():
			// Use a background context for OnStop so it can do cleanup
			// even though the agent context is cancelled. (#24)
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			entry.handler.OnStop(stopCtx, entry.ref)
			stopCancel()
			return
		case msg, ok := <-entry.mailbox:
			if !ok {
				stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
				entry.handler.OnStop(stopCtx, entry.ref)
				stopCancel()
				return
			}
			msgCtx, msgSpan := tracer.Start(entry.ctx, "actor.message_handle",
				trace.WithAttributes(
					attribute.String("agent.id", entry.ref.ID),
					attribute.String("message.type", messageTypeName(msg.Type)),
				),
			)
			err := entry.handler.HandleMessage(msgCtx, entry.ref, msg)
			if err != nil {
				msgSpan.RecordError(err)
				msgSpan.End()
				s.handleFailure(entry, err)
				return
			}
			msgSpan.End()
			s.msgsHandled.Add(1)
		}
	}
}

func (s *agentSupervisor) handleFailure(entry *agentEntry, err error) {
	_, span := tracer.Start(entry.ctx, "actor.failure",
		trace.WithAttributes(
			attribute.String("agent.id", entry.ref.ID),
			attribute.String("error", err.Error()),
		),
	)
	defer span.End()
	span.RecordError(err)

	s.totalFailed.Add(1)

	strategy := entry.strategy
	now := time.Now()

	// Clean old restarts outside the window.
	windowStart := now.Add(-strategy.RestartWindow)
	var recent []time.Time
	for _, t := range entry.restarts {
		if t.After(windowStart) {
			recent = append(recent, t)
		}
	}
	entry.restarts = recent

	if len(entry.restarts) >= strategy.MaxRestarts {
		// 1.2: ExhaustionPolicy — switch on configured policy instead of always stopping.
		switch strategy.OnExhausted {
		case PolicyEscalate:
			slog.Error("actor: agent exhausted restarts, escalating",
				"agent", entry.ref.ID,
				"restarts", len(entry.restarts),
				"max", strategy.MaxRestarts,
				"error", err,
			)
			// Note: totalFailed already incremented at top of handleFailure.
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			entry.handler.OnStop(stopCtx, entry.ref)
			stopCancel()
			return
		case PolicyLogAndContinue:
			slog.Warn("actor: agent exhausted restarts, resetting counter and continuing",
				"agent", entry.ref.ID,
				"restarts", len(entry.restarts),
				"max", strategy.MaxRestarts,
			)
			// Reset restart counter so the agent can keep running.
			entry.restarts = nil
			// Fall through to restart logic below.
		default: // PolicyStop
			slog.Info("actor: agent exhausted restarts, stopping",
				"agent", entry.ref.ID,
				"restarts", len(entry.restarts),
				"max", strategy.MaxRestarts,
			)
			stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
			entry.handler.OnStop(stopCtx, entry.ref)
			stopCancel()
			return
		}
	}

	// Mark as restarting so Send() returns an error during backoff. (#4)
	entry.restarting.Store(true)

	// Drain mailbox before backoff to avoid losing messages silently. (#4)
	for {
		select {
		case <-entry.mailbox:
		default:
			goto drained
		}
	}
drained:

	// Restart with backoff.
	entry.restarts = append(entry.restarts, now)
	s.totalRestarted.Add(1)
	if restartCounter != nil {
		restartCounter.Add(entry.ctx, 1, metric.WithAttributes(attribute.String("agent.id", entry.ref.ID)))
	}

	backoff := strategy.Backoff.Initial
	for range len(entry.restarts) - 1 {
		backoff = time.Duration(float64(backoff) * strategy.Backoff.Multiplier)
		if backoff > strategy.Backoff.Max {
			backoff = strategy.Backoff.Max
			break
		}
	}

	select {
	case <-time.After(backoff):
	case <-entry.ctx.Done():
		return
	}

	// Allocate a new done channel for the new goroutine.
	entry.done = make(chan struct{})
	go s.runAgent(entry)
}

func (s *agentSupervisor) Send(_ context.Context, ref *AgentRef, msg Message) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.agents[ref.ID]
	if !ok {
		return fmt.Errorf("actor: agent %q not found", ref.ID)
	}

	// Reject sends while the agent is restarting. (#4)
	if entry.restarting.Load() {
		return fmt.Errorf("actor: agent %q is restarting", ref.ID)
	}

	select {
	case entry.mailbox <- msg:
		return nil
	default:
		return fmt.Errorf("actor: agent %q mailbox full", ref.ID)
	}
}

func (s *agentSupervisor) Request(ctx context.Context, ref *AgentRef, msg Message, timeout time.Duration) (Message, error) {
	replyCh := make(chan Message, 1)
	msg.ReplyTo = replyCh
	msg.Type = MessageRequest

	if err := s.Send(ctx, ref, msg); err != nil {
		return Message{}, err
	}

	select {
	case reply := <-replyCh:
		return reply, nil
	case <-time.After(timeout):
		return Message{}, fmt.Errorf("actor: request to %q timed out after %v", ref.ID, timeout)
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
}

func (s *agentSupervisor) SetStrategy(ref *AgentRef, strategy SupervisionStrategy) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.agents[ref.ID]
	if !ok {
		return fmt.Errorf("actor: agent %q not found", ref.ID)
	}
	entry.strategy = strategy
	return nil
}

func (s *agentSupervisor) Shutdown(_ context.Context, ref *AgentRef) error {
	s.mu.Lock()
	entry, ok := s.agents[ref.ID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("actor: agent %q not found", ref.ID)
	}

	// Capture done before cancellation — cancel may trigger handleFailure which
	// reassigns entry.done for a new restart goroutine.
	doneCh := entry.done
	entry.cancel()
	close(entry.mailbox) // close mailbox so goroutine unblocks (#16, #17)
	// 1.4: Do NOT delete from map yet — unlock first to avoid holding lock while waiting.
	s.mu.Unlock()

	// Wait for the goroutine to exit with a timeout. (#10)
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
	}

	// 1.4: Delete AFTER goroutine exits to prevent race with concurrent List().
	s.mu.Lock()
	delete(s.agents, ref.ID)
	s.mu.Unlock()

	if activeGauge != nil {
		activeGauge.Add(context.Background(), -1, metric.WithAttributes(attribute.String("agent.id", ref.ID)))
	}
	return nil
}

func (s *agentSupervisor) List(_ context.Context) ([]*AgentRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	refs := make([]*AgentRef, 0, len(s.agents))
	for _, entry := range s.agents {
		refs = append(refs, entry.ref)
	}
	return refs, nil
}

func (s *agentSupervisor) Stats() SupervisorStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return SupervisorStats{
		ActiveAgents:    len(s.agents),
		TotalSpawned:    int(s.totalSpawned.Load()),
		TotalRestarted:  int(s.totalRestarted.Load()),
		TotalFailed:     int(s.totalFailed.Load()),
		MessagesHandled: s.msgsHandled.Load(),
	}
}

func (s *agentSupervisor) Close() error {
	s.mu.Lock()

	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	// Collect done channels before signalling goroutines to stop.
	var doneChans []chan struct{}
	for _, entry := range s.agents {
		doneChans = append(doneChans, entry.done)
		entry.cancel()
		close(entry.mailbox) // unblock goroutines (#16)
		if activeGauge != nil {
			activeGauge.Add(context.Background(), -1, metric.WithAttributes(attribute.String("agent.id", entry.ref.ID)))
		}
	}
	s.agents = nil
	s.mu.Unlock()

	// Wait for all goroutines to exit (with timeout).
	for _, ch := range doneChans {
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
		}
	}
	return nil
}
