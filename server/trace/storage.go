package trace

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TraceStorage struct {
	db *sql.DB
}

type Trace struct {
	TraceID    string     `json:"trace_id"`
	SessionID  string     `json:"session_id"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	RootSpanID string     `json:"root_span_id,omitempty"`
	Status     string     `json:"status"`
}

type Span struct {
	SpanID    string                 `json:"span_id"`
	TraceID   string                 `json:"trace_id"`
	ParentID  string                 `json:"parent_id,omitempty"`
	Name      string                 `json:"name"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Inputs    map[string]interface{} `json:"inputs,omitempty"`
	Outputs   map[string]interface{} `json:"outputs,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Status    string                 `json:"status"`
}

func NewTraceStorage(db *sql.DB) (*TraceStorage, error) {
	ts := &TraceStorage{
		db: db,
	}

	if err := ts.initTraceSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize trace schema: %w", err)
	}

	return ts, nil
}

func (ts *TraceStorage) initTraceSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS traces (
		trace_id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		root_span_id TEXT,
		status TEXT DEFAULT 'active'
	);

	-- Required indexes per spec Section 5.2
	CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(session_id);
	CREATE INDEX IF NOT EXISTS idx_traces_time ON traces(start_time DESC);
	
	-- Additional index: Enables filtering by status (active/completed/failed) for trace replay and monitoring
	CREATE INDEX IF NOT EXISTS idx_traces_status ON traces(status);

	CREATE TABLE IF NOT EXISTS spans (
		span_id TEXT PRIMARY KEY,
		trace_id TEXT NOT NULL,
		parent_id TEXT,
		name TEXT NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		inputs TEXT,
		outputs TEXT,
		metadata TEXT,
		status TEXT DEFAULT 'running',
		FOREIGN KEY (trace_id) REFERENCES traces(trace_id) ON DELETE CASCADE
	);

	-- Required indexes per spec Section 5.2
	CREATE INDEX IF NOT EXISTS idx_spans_trace ON spans(trace_id);
	CREATE INDEX IF NOT EXISTS idx_spans_parent ON spans(parent_id);
	CREATE INDEX IF NOT EXISTS idx_spans_time ON spans(start_time DESC);
	
	-- Additional index: Enables filtering spans by operation type (intent_classification, tool_execution, etc.)
	CREATE INDEX IF NOT EXISTS idx_spans_name ON spans(name);
	
	-- Additional index: Enables filtering by status (running/completed/failed) for debugging and monitoring
	CREATE INDEX IF NOT EXISTS idx_spans_status ON spans(status);
	`

	_, err := ts.db.Exec(schema)
	return err
}

func (ts *TraceStorage) StoreTrace(ctx context.Context, trace *Trace) error {
	if trace.TraceID == "" {
		trace.TraceID = uuid.New().String()
	}

	if trace.Status == "" {
		trace.Status = "active"
	}

	query := `
		INSERT INTO traces (trace_id, session_id, start_time, end_time, root_span_id, status)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(trace_id) DO UPDATE SET
			end_time = excluded.end_time,
			root_span_id = excluded.root_span_id,
			status = excluded.status
	`

	_, err := ts.db.ExecContext(ctx, query,
		trace.TraceID,
		trace.SessionID,
		trace.StartTime,
		trace.EndTime,
		trace.RootSpanID,
		trace.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to store trace: %w", err)
	}

	return nil
}

func (ts *TraceStorage) RetrieveTrace(ctx context.Context, traceID string) (*Trace, error) {
	query := `SELECT trace_id, session_id, start_time, end_time, root_span_id, status FROM traces WHERE trace_id = ?`

	var trace Trace
	var endTime sql.NullTime

	err := ts.db.QueryRowContext(ctx, query, traceID).Scan(
		&trace.TraceID,
		&trace.SessionID,
		&trace.StartTime,
		&endTime,
		&trace.RootSpanID,
		&trace.Status,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve trace: %w", err)
	}

	if endTime.Valid {
		trace.EndTime = &endTime.Time
	}

	return &trace, nil
}

func (ts *TraceStorage) ListTraces(ctx context.Context, sessionID string, limit int) ([]Trace, error) {
	query := `
		SELECT trace_id, session_id, start_time, end_time, root_span_id, status
		FROM traces
		WHERE session_id = ?
		ORDER BY start_time DESC
		LIMIT ?
	`

	rows, err := ts.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list traces: %w", err)
	}
	defer rows.Close()

	traces := []Trace{}
	for rows.Next() {
		var trace Trace
		var endTime sql.NullTime

		err := rows.Scan(
			&trace.TraceID,
			&trace.SessionID,
			&trace.StartTime,
			&endTime,
			&trace.RootSpanID,
			&trace.Status,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan trace row: %w", err)
		}

		if endTime.Valid {
			trace.EndTime = &endTime.Time
		}

		traces = append(traces, trace)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trace rows: %w", err)
	}

	return traces, nil
}

func (ts *TraceStorage) StoreSpan(ctx context.Context, span *Span) error {
	if span.SpanID == "" {
		span.SpanID = uuid.New().String()
	}

	if span.Status == "" {
		span.Status = "running"
	}

	var inputsJSON, outputsJSON, metadataJSON []byte
	var err error

	if span.Inputs != nil {
		inputsJSON, err = json.Marshal(span.Inputs)
		if err != nil {
			return fmt.Errorf("failed to marshal inputs: %w", err)
		}
	}

	if span.Outputs != nil {
		outputsJSON, err = json.Marshal(span.Outputs)
		if err != nil {
			return fmt.Errorf("failed to marshal outputs: %w", err)
		}
	}

	if span.Metadata != nil {
		metadataJSON, err = json.Marshal(span.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO spans (span_id, trace_id, parent_id, name, start_time, end_time, inputs, outputs, metadata, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(span_id) DO UPDATE SET
			end_time = excluded.end_time,
			outputs = excluded.outputs,
			metadata = excluded.metadata,
			status = excluded.status
	`

	_, err = ts.db.ExecContext(ctx, query,
		span.SpanID,
		span.TraceID,
		span.ParentID,
		span.Name,
		span.StartTime,
		span.EndTime,
		string(inputsJSON),
		string(outputsJSON),
		string(metadataJSON),
		span.Status,
	)

	if err != nil {
		return fmt.Errorf("failed to store span: %w", err)
	}

	return nil
}

func (ts *TraceStorage) RetrieveSpan(ctx context.Context, spanID string) (*Span, error) {
	query := `SELECT span_id, trace_id, parent_id, name, start_time, end_time, inputs, outputs, metadata, status FROM spans WHERE span_id = ?`

	var span Span
	var endTime sql.NullTime
	var inputsJSON, outputsJSON, metadataJSON sql.NullString

	err := ts.db.QueryRowContext(ctx, query, spanID).Scan(
		&span.SpanID,
		&span.TraceID,
		&span.ParentID,
		&span.Name,
		&span.StartTime,
		&endTime,
		&inputsJSON,
		&outputsJSON,
		&metadataJSON,
		&span.Status,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("span not found: %s", spanID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to retrieve span: %w", err)
	}

	if endTime.Valid {
		span.EndTime = &endTime.Time
	}

	if inputsJSON.Valid && inputsJSON.String != "" {
		if err := json.Unmarshal([]byte(inputsJSON.String), &span.Inputs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal inputs: %w", err)
		}
	}

	if outputsJSON.Valid && outputsJSON.String != "" {
		if err := json.Unmarshal([]byte(outputsJSON.String), &span.Outputs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal outputs: %w", err)
		}
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &span.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &span, nil
}

func (ts *TraceStorage) ListSpansByTrace(ctx context.Context, traceID string) ([]Span, error) {
	query := `
		SELECT span_id, trace_id, parent_id, name, start_time, end_time, inputs, outputs, metadata, status
		FROM spans
		WHERE trace_id = ?
		ORDER BY start_time ASC
	`

	rows, err := ts.db.QueryContext(ctx, query, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list spans: %w", err)
	}
	defer rows.Close()

	spans := []Span{}
	for rows.Next() {
		var span Span
		var endTime sql.NullTime
		var inputsJSON, outputsJSON, metadataJSON sql.NullString

		err := rows.Scan(
			&span.SpanID,
			&span.TraceID,
			&span.ParentID,
			&span.Name,
			&span.StartTime,
			&endTime,
			&inputsJSON,
			&outputsJSON,
			&metadataJSON,
			&span.Status,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan span row: %w", err)
		}

		if endTime.Valid {
			span.EndTime = &endTime.Time
		}

		if inputsJSON.Valid && inputsJSON.String != "" {
			if err := json.Unmarshal([]byte(inputsJSON.String), &span.Inputs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal inputs for span %s: %w", span.SpanID, err)
			}
		}

		if outputsJSON.Valid && outputsJSON.String != "" {
			if err := json.Unmarshal([]byte(outputsJSON.String), &span.Outputs); err != nil {
				return nil, fmt.Errorf("failed to unmarshal outputs for span %s: %w", span.SpanID, err)
			}
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &span.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata for span %s: %w", span.SpanID, err)
			}
		}

		spans = append(spans, span)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating span rows: %w", err)
	}

	return spans, nil
}

func (ts *TraceStorage) UpdateTraceStatus(ctx context.Context, traceID string, status string, endTime time.Time) error {
	query := `UPDATE traces SET status = ?, end_time = ? WHERE trace_id = ?`

	_, err := ts.db.ExecContext(ctx, query, status, endTime, traceID)
	if err != nil {
		return fmt.Errorf("failed to update trace status: %w", err)
	}

	return nil
}

func (ts *TraceStorage) UpdateSpanStatus(ctx context.Context, spanID string, status string, endTime time.Time) error {
	query := `UPDATE spans SET status = ?, end_time = ? WHERE span_id = ?`

	_, err := ts.db.ExecContext(ctx, query, status, endTime, spanID)
	if err != nil {
		return fmt.Errorf("failed to update span status: %w", err)
	}

	return nil
}
