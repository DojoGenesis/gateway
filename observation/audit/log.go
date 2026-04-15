package audit

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// AuditLog records and queries capability audit entries.
type AuditLog interface {
	// Record appends an audit entry to the log.
	Record(ctx context.Context, entry AuditEntry) error

	// Query retrieves audit entries matching the filter.
	Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error)

	// Export writes matching entries in the specified format.
	Export(ctx context.Context, filter AuditFilter, format ExportFormat, w io.Writer) error

	// Close releases audit log resources.
	Close() error
}

// sqliteAuditLog is the SQLite-backed audit log implementation.
type sqliteAuditLog struct {
	db *sql.DB
}

// NewSQLiteAuditLog creates a new SQLite-backed audit log.
func NewSQLiteAuditLog(dbPath string) (AuditLog, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("audit: open db: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-4000",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("audit: pragma %q: %w", p, err)
		}
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := createAuditTables(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &sqliteAuditLog{db: db}, nil
}

func createAuditTables(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS audit_log (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		agent_id TEXT NOT NULL,
		action TEXT NOT NULL,
		tool TEXT,
		tool_args TEXT,
		capabilities_granted TEXT,
		result_hash TEXT,
		duration_ns INTEGER,
		metadata TEXT
	)`)
	if err != nil {
		return fmt.Errorf("audit: create table: %w", err)
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_audit_agent ON audit_log(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_tool ON audit_log(tool)`,
	}
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf("audit: create index: %w", err)
		}
	}
	return nil
}

func (l *sqliteAuditLog) Record(ctx context.Context, entry AuditEntry) error {
	if entry.ID == "" {
		return fmt.Errorf("audit: entry ID is required")
	}

	// Check marshal errors instead of silently discarding them. (#5)
	toolArgsJSON, err := json.Marshal(entry.ToolArgs)
	if err != nil {
		return fmt.Errorf("audit: marshal tool args: %w", err)
	}
	capsJSON, err := json.Marshal(entry.CapabilitiesGranted)
	if err != nil {
		return fmt.Errorf("audit: marshal capabilities: %w", err)
	}
	metaJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return fmt.Errorf("audit: marshal metadata: %w", err)
	}

	_, err = l.db.ExecContext(ctx,
		`INSERT INTO audit_log (id, timestamp, agent_id, action, tool, tool_args, capabilities_granted, result_hash, duration_ns, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.ID,
		entry.Timestamp,
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
		return fmt.Errorf("audit: record: %w", err)
	}
	return nil
}

func (l *sqliteAuditLog) Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
	query := `SELECT id, timestamp, agent_id, action, tool, tool_args, capabilities_granted, result_hash, duration_ns, metadata FROM audit_log`
	var conditions []string
	var args []interface{}

	if filter.AgentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, filter.AgentID)
	}
	if len(filter.Actions) > 0 {
		placeholders := make([]string, len(filter.Actions))
		for i, a := range filter.Actions {
			placeholders[i] = "?"
			args = append(args, string(a))
		}
		conditions = append(conditions, "action IN ("+strings.Join(placeholders, ",")+")")
	}
	if !filter.From.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, filter.To)
	}
	if filter.Tool != "" {
		conditions = append(conditions, "tool = ?")
		args = append(args, filter.Tool)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	// Support offset for pagination. (#26)
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var action string
		var toolArgsJSON, capsJSON, metaJSON sql.NullString
		var durationNs int64

		if err := rows.Scan(&e.ID, &e.Timestamp, &e.AgentID, &action, &e.Tool,
			&toolArgsJSON, &capsJSON, &e.ResultHash, &durationNs, &metaJSON); err != nil {
			return nil, fmt.Errorf("audit: query scan: %w", err)
		}
		e.Action = Action(action)
		e.Duration = time.Duration(durationNs)

		if toolArgsJSON.Valid && toolArgsJSON.String != "" {
			_ = json.Unmarshal([]byte(toolArgsJSON.String), &e.ToolArgs)
		}
		if capsJSON.Valid && capsJSON.String != "" {
			_ = json.Unmarshal([]byte(capsJSON.String), &e.CapabilitiesGranted)
		}
		if metaJSON.Valid && metaJSON.String != "" {
			_ = json.Unmarshal([]byte(metaJSON.String), &e.Metadata)
		}

		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (l *sqliteAuditLog) Export(ctx context.Context, filter AuditFilter, format ExportFormat, w io.Writer) error {
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
		// Check Write errors instead of ignoring return values. (#19)
		if err := cw.Write([]string{"id", "timestamp", "agent_id", "action", "tool", "result_hash", "duration_ms"}); err != nil {
			return fmt.Errorf("audit: csv header: %w", err)
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
				return fmt.Errorf("audit: csv row: %w", err)
			}
		}
		return cw.Error()
	default:
		return fmt.Errorf("audit: unsupported export format: %s", format)
	}
}

func (l *sqliteAuditLog) Close() error {
	return l.db.Close()
}
