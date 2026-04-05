package audit

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/runtime/d1client"
)

// D1Config holds connection parameters for the D1-backed audit log.
type D1Config struct {
	// AccountID is the Cloudflare account identifier.
	AccountID string

	// DatabaseID is the D1 database ID (UUID).
	DatabaseID string

	// APIToken is the Cloudflare API token with D1 read/write permissions.
	APIToken string

	// BaseURL overrides the Cloudflare API base URL (used in tests).
	BaseURL string
}

// d1AuditLog is the D1-backed AuditLog implementation.
type d1AuditLog struct {
	client *d1client.Client
}

// NewD1AuditLog creates a D1-backed AuditLog. The remote `audit_log` table must
// already exist (matching the schema in observation/audit/log.go createAuditTables).
func NewD1AuditLog(cfg D1Config) (AuditLog, error) {
	if cfg.AccountID == "" || cfg.DatabaseID == "" || cfg.APIToken == "" {
		return nil, errors.New("audit/d1: AccountID, DatabaseID, and APIToken are required")
	}
	return &d1AuditLog{
		client: d1client.New(d1client.Config{
			AccountID:  cfg.AccountID,
			DatabaseID: cfg.DatabaseID,
			APIToken:   cfg.APIToken,
			BaseURL:    cfg.BaseURL,
		}),
	}, nil
}

// ---------------------------------------------------------------------------
// AuditLog interface implementation
// ---------------------------------------------------------------------------

func (l *d1AuditLog) Record(ctx context.Context, entry AuditEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("audit/d1: entry ID is required")
	}

	toolArgsJSON, err := json.Marshal(entry.ToolArgs)
	if err != nil {
		return fmt.Errorf("audit/d1: marshal tool_args: %w", err)
	}
	capsJSON, err := json.Marshal(entry.CapabilitiesGranted)
	if err != nil {
		return fmt.Errorf("audit/d1: marshal capabilities: %w", err)
	}
	metaJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("audit/d1: marshal metadata: %w", err)
	}

	_, err = l.client.Exec(ctx,
		`INSERT INTO audit_log
		 (id, timestamp, agent_id, action, tool, tool_args, capabilities_granted, result_hash, duration_ns, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		entry.AgentID,
		string(entry.Action),
		entry.Tool,
		string(toolArgsJSON),
		string(capsJSON),
		entry.ResultHash,
		entry.Duration.Nanoseconds(),
		string(metaJSON),
	)
	if err != nil {
		return fmt.Errorf("audit/d1: record: %w", err)
	}
	return nil
}

func (l *d1AuditLog) Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	sql := `SELECT id, timestamp, agent_id, action, tool, tool_args, capabilities_granted, result_hash, duration_ns, metadata
	        FROM audit_log`

	var conditions []string
	var args []any

	if filter.AgentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, filter.AgentID)
	}
	if len(filter.Actions) > 0 {
		placeholders := strings.Repeat("?,", len(filter.Actions))
		placeholders = placeholders[:len(placeholders)-1]
		conditions = append(conditions, "action IN ("+placeholders+")")
		for _, a := range filter.Actions {
			args = append(args, string(a))
		}
	}
	if !filter.From.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, filter.From.UTC().Format(time.RFC3339Nano))
	}
	if !filter.To.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, filter.To.UTC().Format(time.RFC3339Nano))
	}
	if filter.Tool != "" {
		conditions = append(conditions, "tool = ?")
		args = append(args, filter.Tool)
	}
	if len(conditions) > 0 {
		sql += " WHERE " + strings.Join(conditions, " AND ")
	}
	sql += " ORDER BY timestamp DESC"
	if filter.Limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	if filter.Offset > 0 {
		sql += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := l.client.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("audit/d1: query: %w", err)
	}

	entries := make([]AuditEntry, 0, len(rows))
	for _, row := range rows {
		e, err := rowToEntry(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (l *d1AuditLog) Export(ctx context.Context, filter AuditFilter, format ExportFormat, w io.Writer) error {
	entries, err := l.Query(ctx, filter)
	if err != nil {
		return err
	}

	switch format {
	case ExportJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	case ExportCSV:
		cw := csv.NewWriter(w)
		defer cw.Flush()
		if err := cw.Write([]string{"id", "timestamp", "agent_id", "action", "tool", "result_hash", "duration_ms"}); err != nil {
			return fmt.Errorf("audit/d1: csv header: %w", err)
		}
		for _, e := range entries {
			if err := cw.Write([]string{
				e.ID,
				e.Timestamp.Format(time.RFC3339),
				e.AgentID,
				string(e.Action),
				e.Tool,
				e.ResultHash,
				fmt.Sprintf("%.2f", float64(e.Duration.Milliseconds())),
			}); err != nil {
				return fmt.Errorf("audit/d1: csv row: %w", err)
			}
		}
		return cw.Error()
	default:
		return fmt.Errorf("audit/d1: unsupported export format: %s", format)
	}
}

// Close is a no-op; D1 uses stateless HTTP with no persistent connection.
func (l *d1AuditLog) Close() error { return nil }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func rowToEntry(row map[string]any) (AuditEntry, error) {
	var e AuditEntry
	e.ID = d1client.String(row["id"])
	e.AgentID = d1client.String(row["agent_id"])
	e.Action = Action(d1client.String(row["action"]))
	e.Tool = d1client.String(row["tool"])
	e.ResultHash = d1client.String(row["result_hash"])
	e.Duration = time.Duration(d1client.Int64(row["duration_ns"]))

	tsStr := d1client.String(row["timestamp"])
	if tsStr != "" {
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			// Try RFC3339 without nanoseconds.
			ts, err = time.Parse(time.RFC3339, tsStr)
			if err != nil {
				return AuditEntry{}, fmt.Errorf("audit/d1: parse timestamp %q: %w", tsStr, err)
			}
		}
		e.Timestamp = ts.UTC()
	}

	if s := d1client.String(row["tool_args"]); s != "" && s != "null" {
		json.Unmarshal([]byte(s), &e.ToolArgs) //nolint:errcheck
	}
	if s := d1client.String(row["capabilities_granted"]); s != "" && s != "null" {
		json.Unmarshal([]byte(s), &e.CapabilitiesGranted) //nolint:errcheck
	}
	if s := d1client.String(row["metadata"]); s != "" && s != "null" {
		json.Unmarshal([]byte(s), &e.Metadata) //nolint:errcheck
	}
	return e, nil
}
