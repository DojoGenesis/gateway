package services

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/DojoGenesis/gateway/events"
	"github.com/google/uuid"
)

const (
	tapBufferSize  = 1000
	tapBatchSize   = 50
	tapFlushPeriod = 5 * time.Second
	tapHTTPTimeout = 10 * time.Second
)

// telemetryEvent mirrors the Worker's TelemetryEvent interface.
type telemetryEvent struct {
	Type string                 `json:"type"`
	Ts   int64                  `json:"ts"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// ingestPayload mirrors the Worker's IngestPayload interface.
type ingestPayload struct {
	SessionID string           `json:"session_id"`
	Events    []telemetryEvent `json:"events"`
}

// TelemetryTap is a non-blocking event sink that batches StreamEvents and
// forwards them to an external telemetry Worker via POST /api/telemetry/ingest.
//
// It is safe for concurrent use. Push() never blocks the caller.
// If the buffer is full, events are dropped with a warning (no backpressure).
type TelemetryTap struct {
	workerURL string
	sessionID string
	ch        chan events.StreamEvent
	stopCh    chan struct{}
	wg        sync.WaitGroup
	client    *http.Client
	once      sync.Once
}

// NewTelemetryTap creates a TelemetryTap that forwards events to workerURL.
// Call Start() to begin the background flush goroutine.
func NewTelemetryTap(workerURL string) *TelemetryTap {
	return &TelemetryTap{
		workerURL: workerURL,
		sessionID: uuid.New().String(),
		ch:        make(chan events.StreamEvent, tapBufferSize),
		stopCh:    make(chan struct{}),
		client:    &http.Client{Timeout: tapHTTPTimeout},
	}
}

// Start begins the background goroutine. Idempotent.
func (t *TelemetryTap) Start() {
	t.once.Do(func() {
		t.wg.Add(1)
		go t.run()
		slog.Info("telemetry tap started", "session_id", t.sessionID, "worker_url", t.workerURL)
	})
}

// Stop signals the background goroutine to flush remaining events and exit.
// It blocks until the goroutine exits.
func (t *TelemetryTap) Stop() {
	close(t.stopCh)
	t.wg.Wait()
}

// Push queues an event for forwarding. Non-blocking: if the buffer is full,
// the event is dropped with a warning log.
func (t *TelemetryTap) Push(evt events.StreamEvent) {
	select {
	case t.ch <- evt:
	default:
		slog.Warn("telemetry tap buffer full, dropping event", "type", evt.Type)
	}
}

func (t *TelemetryTap) run() {
	defer t.wg.Done()
	ticker := time.NewTicker(tapFlushPeriod)
	defer ticker.Stop()

	var batch []events.StreamEvent

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toSend := batch
		batch = nil
		t.post(toSend)
	}

	for {
		select {
		case <-t.stopCh:
			// Drain remaining buffered events before exiting.
		drain:
			for {
				select {
				case evt := <-t.ch:
					batch = append(batch, evt)
				default:
					break drain
				}
			}
			flush()
			return
		case <-ticker.C:
			flush()
		case evt := <-t.ch:
			batch = append(batch, evt)
			if len(batch) >= tapBatchSize {
				flush()
			}
		}
	}
}

func (t *TelemetryTap) post(batch []events.StreamEvent) {
	telEvents := make([]telemetryEvent, 0, len(batch))
	for _, evt := range batch {
		telEvents = append(telEvents, telemetryEvent{
			Type: string(evt.Type),
			Ts:   evt.Timestamp.Unix(),
			Data: evt.Data,
		})
	}

	payload := ingestPayload{
		SessionID: t.sessionID,
		Events:    telEvents,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("telemetry tap: marshal failed", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), tapHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.workerURL+"/api/telemetry/ingest", bytes.NewReader(body))
	if err != nil {
		slog.Warn("telemetry tap: build request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		slog.Warn("telemetry tap: post failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Warn("telemetry tap: worker returned error", "status", resp.StatusCode)
	} else {
		slog.Debug("telemetry tap: flushed events", "count", len(batch))
	}
}
