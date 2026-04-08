package services

import (
	"sync"
	"time"
)

// ProviderLatencyDataPoint is a single recorded data point for provider latency.
type ProviderLatencyDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	LatencyMs int64     `json:"latency_ms"`
	Status    string    `json:"status"` // "ok" or "error"
	ErrorRate float64   `json:"error_rate"`
}

// ProviderLatencyHistory stores the last N data points for a single provider.
type ProviderLatencyHistory struct {
	Name       string                     `json:"name"`
	DataPoints []ProviderLatencyDataPoint `json:"data_points"`
}

// ProviderLatencyTracker is an in-memory ring buffer that records latency
// data points per provider after each LLM invocation.
type ProviderLatencyTracker struct {
	mu          sync.RWMutex
	maxPoints   int
	buffers     map[string]*ringBuffer
	errorCounts map[string]*errorCounter
}

type ringBuffer struct {
	points []ProviderLatencyDataPoint
	head   int
	count  int
}

type errorCounter struct {
	total  int64
	errors int64
}

// NewProviderLatencyTracker creates a tracker that stores the last maxPoints
// data points per provider.
func NewProviderLatencyTracker(maxPoints int) *ProviderLatencyTracker {
	if maxPoints <= 0 {
		maxPoints = 60
	}
	return &ProviderLatencyTracker{
		maxPoints:   maxPoints,
		buffers:     make(map[string]*ringBuffer),
		errorCounts: make(map[string]*errorCounter),
	}
}

// Record stores a latency data point for a provider.
func (t *ProviderLatencyTracker) Record(providerName string, latencyMs int64, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	buf, ok := t.buffers[providerName]
	if !ok {
		buf = &ringBuffer{
			points: make([]ProviderLatencyDataPoint, t.maxPoints),
		}
		t.buffers[providerName] = buf
	}

	ec, ok := t.errorCounts[providerName]
	if !ok {
		ec = &errorCounter{}
		t.errorCounts[providerName] = ec
	}

	ec.total++
	if isError {
		ec.errors++
	}

	// Calculate error rate from the last 100 requests (sliding window approximation)
	errorRate := float64(0)
	if ec.total > 0 {
		errorRate = float64(ec.errors) / float64(ec.total)
	}

	status := "ok"
	if isError {
		status = "error"
	}

	dp := ProviderLatencyDataPoint{
		Timestamp: time.Now(),
		LatencyMs: latencyMs,
		Status:    status,
		ErrorRate: errorRate,
	}

	// Ring buffer insert
	idx := (buf.head + buf.count) % t.maxPoints
	if buf.count < t.maxPoints {
		buf.count++
	} else {
		buf.head = (buf.head + 1) % t.maxPoints
	}
	buf.points[idx] = dp
}

// GetHistory returns the latency history for a specific provider.
// The points parameter limits how many data points to return (0 = all).
func (t *ProviderLatencyTracker) GetHistory(providerName string, points int) *ProviderLatencyHistory {
	t.mu.RLock()
	defer t.mu.RUnlock()

	buf, ok := t.buffers[providerName]
	if !ok {
		return &ProviderLatencyHistory{
			Name:       providerName,
			DataPoints: []ProviderLatencyDataPoint{},
		}
	}

	if points <= 0 || points > buf.count {
		points = buf.count
	}

	// Extract points from ring buffer in chronological order
	result := make([]ProviderLatencyDataPoint, 0, points)
	start := buf.head
	if buf.count > points {
		start = (buf.head + buf.count - points) % t.maxPoints
	}

	for i := 0; i < points; i++ {
		idx := (start + i) % t.maxPoints
		result = append(result, buf.points[idx])
	}

	return &ProviderLatencyHistory{
		Name:       providerName,
		DataPoints: result,
	}
}

// ListProviders returns the names of all providers that have recorded data.
func (t *ProviderLatencyTracker) ListProviders() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	names := make([]string, 0, len(t.buffers))
	for name := range t.buffers {
		names = append(names, name)
	}
	return names
}
