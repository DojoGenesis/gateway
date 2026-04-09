package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Package-level tracer for distributed tracing.
var tracer = otel.Tracer("dojo.runtime.event")

// Package-level meter for metrics.
var meter = otel.Meter("dojo.runtime.event")

// Metric instruments initialized once at package load.
var (
	publishedCounter    metric.Int64Counter
	subscriptionsGauge  metric.Int64UpDownCounter
)

func init() {
	var err error
	publishedCounter, err = meter.Int64Counter("dojo.event.published",
		metric.WithDescription("Number of events published through the bus"),
		metric.WithUnit("{event}"),
	)
	if err != nil {
		publishedCounter, _ = meter.Int64Counter("dojo.event.published")
	}

	subscriptionsGauge, err = meter.Int64UpDownCounter("dojo.event.subscriptions",
		metric.WithDescription("Number of active event subscriptions"),
		metric.WithUnit("{subscription}"),
	)
	if err != nil {
		subscriptionsGauge, _ = meter.Int64UpDownCounter("dojo.event.subscriptions")
	}
}

// Event represents a platform event (CloudEvents-compatible).
type Event struct {
	// ID is a unique event identifier.
	ID string

	// Type is the event type (e.g., "dojo.tool.completed").
	Type string

	// Source identifies the event producer (e.g., "agent/scout-1").
	Source string

	// Subject is the event subject (e.g., "workflow/abc123").
	Subject string

	// Data is the event payload as bytes.
	Data []byte

	// Time is when the event occurred.
	Time time.Time
}

// EventHandler processes a single event.
type EventHandler func(ctx context.Context, event Event) error

// BusStats provides metrics about the event bus.
type BusStats struct {
	EventsPublished   int64
	EventsDelivered   int64
	ActiveSubscribers int
}

// Bus is the event bus interface backed by embedded NATS + JetStream.
type Bus interface {
	// Publish sends an event to all matching subscribers.
	Publish(ctx context.Context, event Event) error

	// Subscribe registers a handler for events matching the filter.
	Subscribe(ctx context.Context, filter EventFilter, handler EventHandler) (Subscription, error)

	// SubscribeDurable creates a JetStream durable consumer for the given
	// subject. The durable name is used as the consumer name in JetStream,
	// enabling resume-on-restart semantics. Returns an error if JetStream
	// (WAL) is not enabled.
	SubscribeDurable(ctx context.Context, subject, durable string, handler EventHandler) (Subscription, error)

	// Request publishes an event and waits for a correlated response.
	Request(ctx context.Context, event Event, timeout time.Duration) (Event, error)

	// Replay replays events from JetStream matching the filter.
	Replay(ctx context.Context, filter EventFilter, from time.Time) (<-chan Event, error)

	// Stats returns bus metrics.
	Stats() BusStats

	// Close shuts down the bus and NATS server.
	Close() error
}

const streamName = "DOJO_EVENTS"
const subjectPrefix = "dojo"

// natsBus is an event bus backed by embedded NATS + JetStream.
type natsBus struct {
	mu     sync.RWMutex
	server *server.Server
	conn   *nats.Conn
	js     nats.JetStreamContext
	subs   map[string]*natsSub
	config Config
	closed bool

	published atomic.Int64
	delivered atomic.Int64
	nextSubID atomic.Int64
}

type natsSub struct {
	id      string
	sub     *nats.Subscription
	filter  EventFilter
	handler EventHandler
	ctx     context.Context
	cancel  context.CancelFunc
	typeSet map[string]struct{}
	srcSet  map[string]struct{}
	subjSet map[string]struct{}
}

// NewBus creates a new Bus backed by embedded NATS + JetStream.
func NewBus(config Config) (Bus, error) {
	opts := &server.Options{
		DontListen: true,
		JetStream:  config.WAL.Enabled,
		NoLog:      true,
		NoSigs:     true,
	}
	if config.WAL.Enabled && config.WAL.DBPath != "" {
		opts.StoreDir = config.WAL.DBPath
	}

	srv, err := server.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("event: create nats server: %w", err)
	}

	srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		srv.Shutdown()
		return nil, fmt.Errorf("event: nats server not ready")
	}

	conn, err := nats.Connect("", nats.InProcessServer(srv))
	if err != nil {
		srv.Shutdown()
		return nil, fmt.Errorf("event: nats connect: %w", err)
	}

	b := &natsBus{
		server: srv,
		conn:   conn,
		subs:   make(map[string]*natsSub),
		config: config,
	}

	if config.WAL.Enabled {
		js, err := conn.JetStream()
		if err != nil {
			conn.Close()
			srv.Shutdown()
			return nil, fmt.Errorf("event: jetstream: %w", err)
		}
		b.js = js

		retention := config.WAL.Retention
		if retention == 0 {
			retention = 7 * 24 * time.Hour
		}

		_, err = js.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Subjects:  []string{subjectPrefix + ".>"},
			Storage:   nats.MemoryStorage,
			Retention: nats.LimitsPolicy,
			MaxAge:    retention,
		})
		if err != nil {
			conn.Close()
			srv.Shutdown()
			return nil, fmt.Errorf("event: add stream: %w", err)
		}
	}

	return b, nil
}

func (b *natsBus) Publish(ctx context.Context, event Event) error {
	ctx, span := tracer.Start(ctx, "event.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("event.type", event.Type),
			attribute.String("event.source", event.Source),
		),
	)
	defer span.End()

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		span.RecordError(fmt.Errorf("event: bus is closed"))
		return fmt.Errorf("event: bus is closed")
	}
	b.mu.RUnlock()

	ce := toCloudEvent(event)
	data, err := json.Marshal(ce)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("event: marshal cloudevent: %w", err)
	}

	subject := eventTypeToSubject(event.Type)

	if b.js != nil {
		_, err = b.js.Publish(subject, data)
	} else {
		err = b.conn.Publish(subject, data)
	}
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("event: publish: %w", err)
	}

	b.published.Add(1)
	publishedCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event.type", event.Type),
			attribute.String("event.source", event.Source),
		),
	)

	// Deliver to local subscribers synchronously for backward compatibility.
	b.mu.RLock()
	var matched []*natsSub
	for _, sub := range b.subs {
		if matchesFilterSet(event, sub) {
			matched = append(matched, sub)
		}
	}
	b.mu.RUnlock()

	var wg sync.WaitGroup
	for _, sub := range matched {
		select {
		case <-sub.ctx.Done():
			continue
		default:
		}
		wg.Add(1)
		go func(s *natsSub) {
			defer wg.Done()
			if err := s.handler(s.ctx, event); err != nil {
				slog.Warn("event: handler error", "sub", s.id, "error", err)
				return
			}
			b.delivered.Add(1)
		}(sub)
	}
	wg.Wait()

	return nil
}

func (b *natsBus) Subscribe(ctx context.Context, filter EventFilter, handler EventHandler) (Subscription, error) {
	ctx, span := tracer.Start(ctx, "event.subscribe",
		trace.WithAttributes(
			attribute.StringSlice("filter.types", filter.Types),
			attribute.StringSlice("filter.sources", filter.Sources),
			attribute.StringSlice("filter.subjects", filter.Subjects),
		),
	)
	defer span.End()

	if handler == nil {
		span.RecordError(fmt.Errorf("event: handler is required"))
		return nil, fmt.Errorf("event: handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		span.RecordError(fmt.Errorf("event: bus is closed"))
		return nil, fmt.Errorf("event: bus is closed")
	}

	id := fmt.Sprintf("sub-%d", b.nextSubID.Add(1))
	subCtx, cancel := context.WithCancel(ctx)

	sub := &natsSub{
		id:      id,
		filter:  filter,
		handler: handler,
		ctx:     subCtx,
		cancel:  cancel,
		typeSet: toSet(filter.Types),
		srcSet:  toSet(filter.Sources),
		subjSet: toSet(filter.Subjects),
	}
	b.subs[id] = sub

	subscriptionsGauge.Add(ctx, 1)

	return &subscriptionImpl{
		id:  id,
		bus: b,
	}, nil
}

func (b *natsBus) SubscribeDurable(ctx context.Context, subject, durable string, handler EventHandler) (Subscription, error) {
	ctx, span := tracer.Start(ctx, "event.subscribe_durable",
		trace.WithAttributes(
			attribute.String("subject", subject),
			attribute.String("durable", durable),
		),
	)
	defer span.End()

	if handler == nil {
		return nil, fmt.Errorf("event: handler is required")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, fmt.Errorf("event: bus is closed")
	}

	if b.js == nil {
		return nil, fmt.Errorf("event: JetStream (WAL) is not enabled; durable consumers require WAL")
	}

	id := fmt.Sprintf("durable-%s-%d", durable, b.nextSubID.Add(1))
	subCtx, cancel := context.WithCancel(ctx)

	sub := &natsSub{
		id:      id,
		filter:  EventFilter{},
		handler: handler,
		ctx:     subCtx,
		cancel:  cancel,
	}

	jsSub, err := b.js.Subscribe(subject, func(msg *nats.Msg) {
		var ce cloudevents.Event
		if err := json.Unmarshal(msg.Data, &ce); err != nil {
			slog.Warn("event: durable unmarshal error", "durable", durable, "error", err)
			msg.Ack()
			return
		}

		event := fromCloudEvent(ce)
		if err := handler(subCtx, event); err != nil {
			slog.Warn("event: durable handler error", "durable", durable, "error", err)
		} else {
			b.delivered.Add(1)
		}
		msg.Ack()
	}, nats.Durable(durable), nats.ManualAck())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("event: durable subscribe %q on %q: %w", durable, subject, err)
	}

	sub.sub = jsSub
	b.subs[id] = sub
	subscriptionsGauge.Add(ctx, 1)

	return &subscriptionImpl{id: id, bus: b}, nil
}

func (b *natsBus) Request(ctx context.Context, event Event, timeout time.Duration) (Event, error) {
	replyType := event.Type + ".reply"
	replyCh := make(chan Event, 1)

	sub, err := b.Subscribe(ctx, EventFilter{
		Types:    []string{replyType},
		Subjects: []string{event.ID},
	}, func(_ context.Context, reply Event) error {
		select {
		case replyCh <- reply:
		default:
		}
		return nil
	})
	if err != nil {
		return Event{}, fmt.Errorf("event: request subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	if err := b.Publish(ctx, event); err != nil {
		return Event{}, fmt.Errorf("event: request publish: %w", err)
	}

	select {
	case reply := <-replyCh:
		return reply, nil
	case <-time.After(timeout):
		return Event{}, fmt.Errorf("event: request timeout after %v", timeout)
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}

func (b *natsBus) Replay(_ context.Context, filter EventFilter, from time.Time) (<-chan Event, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.js == nil {
		return nil, fmt.Errorf("event: WAL is not enabled")
	}

	typeSet := toSet(filter.Types)
	sourceSet := toSet(filter.Sources)
	subjectSet := toSet(filter.Subjects)

	subject := subjectPrefix + ".>"
	if len(filter.Types) == 1 {
		subject = eventTypeToSubject(filter.Types[0])
	}

	ch := make(chan Event, 100)

	// Use ephemeral consumer (no Durable) to avoid leaked consumers in JetStream.
	opts := []nats.SubOpt{}
	if from.IsZero() {
		opts = append(opts, nats.DeliverAll())
	} else {
		opts = append(opts, nats.StartTime(from))
	}

	sub, err := b.js.Subscribe(subject, func(msg *nats.Msg) {
		var ce cloudevents.Event
		if err := json.Unmarshal(msg.Data, &ce); err != nil {
			msg.Ack()
			return
		}
		event := fromCloudEvent(ce)
		if matchesFilterSets(event, typeSet, sourceSet, subjectSet) {
			ch <- event
		}
		msg.Ack()
	}, opts...)
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("event: replay subscribe: %w", err)
	}

	go func() {
		// Allow pending messages to drain, then clean up.
		time.Sleep(200 * time.Millisecond)
		sub.Drain()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}

func (b *natsBus) Stats() BusStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return BusStats{
		EventsPublished:   b.published.Load(),
		EventsDelivered:   b.delivered.Load(),
		ActiveSubscribers: len(b.subs),
	}
}

func (b *natsBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true

	subCount := int64(len(b.subs))
	for _, sub := range b.subs {
		sub.cancel()
		if sub.sub != nil {
			sub.sub.Unsubscribe()
		}
	}
	b.subs = nil
	if subCount > 0 {
		subscriptionsGauge.Add(context.Background(), -subCount)
	}

	if b.conn != nil {
		b.conn.Close()
	}
	if b.server != nil {
		b.server.Shutdown()
	}
	return nil
}

func (b *natsBus) unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sub, ok := b.subs[id]; ok {
		sub.cancel()
		if sub.sub != nil {
			sub.sub.Unsubscribe()
		}
		delete(b.subs, id)
		subscriptionsGauge.Add(context.Background(), -1)
	}
}

// --- CloudEvents conversion ---

func toCloudEvent(e Event) cloudevents.Event {
	ce := cloudevents.New()
	ce.SetID(e.ID)
	ce.SetType(e.Type)
	if e.Source != "" {
		ce.SetSource(e.Source)
	} else {
		ce.SetSource("dojo-platform")
	}
	ce.SetSubject(e.Subject)
	if e.Time.IsZero() {
		ce.SetTime(time.Now())
	} else {
		ce.SetTime(e.Time)
	}
	if e.Data != nil {
		// Use octet-stream to avoid CloudEvents SDK rejecting non-JSON payloads.
		// Callers may pass arbitrary bytes (not necessarily valid JSON).
		ce.SetData("application/octet-stream", e.Data)
	}
	return ce
}

func fromCloudEvent(ce cloudevents.Event) Event {
	return Event{
		ID:      ce.ID(),
		Type:    ce.Type(),
		Source:  ce.Source(),
		Subject: ce.Subject(),
		Data:    ce.Data(),
		Time:    ce.Time(),
	}
}

func eventTypeToSubject(eventType string) string {
	if eventType == "" {
		return subjectPrefix + ".event"
	}
	if strings.HasPrefix(eventType, subjectPrefix+".") {
		return eventType
	}
	return subjectPrefix + "." + eventType
}

// --- Filtering helpers ---

func toSet(ss []string) map[string]struct{} {
	if len(ss) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}

func matchesFilterSet(e Event, sub *natsSub) bool {
	if sub.typeSet != nil {
		if _, ok := sub.typeSet[e.Type]; !ok {
			return false
		}
	}
	if sub.srcSet != nil {
		if _, ok := sub.srcSet[e.Source]; !ok {
			return false
		}
	}
	if sub.subjSet != nil {
		if _, ok := sub.subjSet[e.Subject]; !ok {
			return false
		}
	}
	return true
}

func matchesFilterSets(e Event, typeSet, sourceSet, subjectSet map[string]struct{}) bool {
	if typeSet != nil {
		if _, ok := typeSet[e.Type]; !ok {
			return false
		}
	}
	if sourceSet != nil {
		if _, ok := sourceSet[e.Source]; !ok {
			return false
		}
	}
	if subjectSet != nil {
		if _, ok := subjectSet[e.Subject]; !ok {
			return false
		}
	}
	return true
}

type subscriptionImpl struct {
	id   string
	bus  *natsBus
	once sync.Once
}

func (s *subscriptionImpl) ID() string { return s.id }

func (s *subscriptionImpl) Unsubscribe() error {
	s.once.Do(func() {
		s.bus.unsubscribe(s.id)
	})
	return nil
}
