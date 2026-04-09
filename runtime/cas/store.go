package cas

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ErrNotFound is returned when content is not found.
var ErrNotFound = errors.New("cas: not found")

// Store is the content-addressable storage engine.
type Store interface {
	// Put stores content and returns its SHA-256 reference.
	Put(ctx context.Context, content []byte, meta ContentMeta) (Ref, error)

	// Get retrieves content by its SHA-256 reference.
	Get(ctx context.Context, ref Ref) ([]byte, ContentMeta, error)

	// Has checks if content exists.
	Has(ctx context.Context, ref Ref) (bool, error)

	// Tag assigns a human-readable name+version to a content reference.
	Tag(ctx context.Context, name string, version string, ref Ref) error

	// Resolve looks up a tag to its content reference.
	Resolve(ctx context.Context, name string, version string) (Ref, error)

	// Untag removes a tag by name and version.
	Untag(ctx context.Context, name string, version string) error

	// List returns all tags matching a prefix.
	List(ctx context.Context, prefix string) ([]TagEntry, error)

	// GC removes content not referenced by any tag.
	GC(ctx context.Context) (GCResult, error)

	// Export writes content to a tar archive.
	Export(ctx context.Context, refs []Ref, w io.Writer) error

	// Import reads content from a tar archive.
	Import(ctx context.Context, r io.Reader) ([]Ref, error)

	// Close releases store resources.
	Close() error
}

// sqliteStore is the SQLite-backed CAS implementation.
type sqliteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed Store.
func NewSQLiteStore(dbPath string) (Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cas: open db: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-8000",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("cas: pragma %q: %w", p, err)
		}
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	if err := createTables(db); err != nil {
		db.Close()
		return nil, err
	}

	return &sqliteStore{db: db}, nil
}

func createTables(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS content (
			hash TEXT PRIMARY KEY,
			data BLOB NOT NULL,
			meta TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			name TEXT NOT NULL,
			version TEXT NOT NULL,
			ref TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY(name, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_ref ON tags(ref)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(name)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("cas: create table: %w", err)
		}
	}
	return nil
}

func computeHash(data []byte) Ref {
	h := sha256.Sum256(data)
	return Ref(hex.EncodeToString(h[:]))
}

func (s *sqliteStore) Put(ctx context.Context, content []byte, meta ContentMeta) (Ref, error) {
	ref := computeHash(content)
	meta.Size = int64(len(content))
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("cas: marshal meta: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO content (hash, data, meta, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(hash) DO UPDATE SET meta = excluded.meta, created_at = excluded.created_at`,
		string(ref), content, string(metaJSON), meta.CreatedAt,
	)
	if err != nil {
		return "", fmt.Errorf("cas: put: %w", err)
	}
	return ref, nil
}

func (s *sqliteStore) Get(ctx context.Context, ref Ref) ([]byte, ContentMeta, error) {
	var data []byte
	var metaJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT data, meta FROM content WHERE hash = ?`, string(ref),
	).Scan(&data, &metaJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ContentMeta{}, ErrNotFound
	}
	if err != nil {
		return nil, ContentMeta{}, fmt.Errorf("cas: get: %w", err)
	}

	var meta ContentMeta
	if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
		return nil, ContentMeta{}, fmt.Errorf("cas: unmarshal meta: %w", err)
	}
	return data, meta, nil
}

func (s *sqliteStore) Has(ctx context.Context, ref Ref) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM content WHERE hash = ?`, string(ref),
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("cas: has: %w", err)
	}
	return count > 0, nil
}

func (s *sqliteStore) Tag(ctx context.Context, name string, version string, ref Ref) error {
	exists, err := s.Has(ctx, ref)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("cas: tag: ref %s not found", ref)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO tags (name, version, ref, created_at) VALUES (?, ?, ?, ?)`,
		name, version, string(ref), time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("cas: tag: %w", err)
	}
	return nil
}

func (s *sqliteStore) Untag(ctx context.Context, name string, version string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM tags WHERE name = ? AND version = ?`, name, version,
	)
	if err != nil {
		return fmt.Errorf("cas: untag: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("cas: untag rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *sqliteStore) Resolve(ctx context.Context, name string, version string) (Ref, error) {
	var ref string
	err := s.db.QueryRowContext(ctx,
		`SELECT ref FROM tags WHERE name = ? AND version = ?`, name, version,
	).Scan(&ref)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("cas: resolve: %w", err)
	}
	return Ref(ref), nil
}

func (s *sqliteStore) List(ctx context.Context, prefix string) ([]TagEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT t.name, t.version, t.ref, c.meta
		 FROM tags t JOIN content c ON t.ref = c.hash
		 WHERE t.name LIKE ?
		 ORDER BY t.name, t.version`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("cas: list: %w", err)
	}
	defer rows.Close()

	var entries []TagEntry
	for rows.Next() {
		var e TagEntry
		var metaJSON string
		if err := rows.Scan(&e.Name, &e.Version, &e.Ref, &metaJSON); err != nil {
			return nil, fmt.Errorf("cas: list scan: %w", err)
		}
		if err := json.Unmarshal([]byte(metaJSON), &e.Meta); err != nil {
			return nil, fmt.Errorf("cas: list unmarshal: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *sqliteStore) GC(ctx context.Context) (GCResult, error) {
	// Use a transaction to prevent race between SELECT and DELETE. (#2)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return GCResult{}, fmt.Errorf("cas: gc begin: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT c.hash, LENGTH(c.data) FROM content c
		 LEFT JOIN tags t ON c.hash = t.ref
		 WHERE t.ref IS NULL`,
	)
	if err != nil {
		return GCResult{}, fmt.Errorf("cas: gc query: %w", err)
	}
	defer rows.Close()

	var hashes []string
	var totalFreed int64
	for rows.Next() {
		var hash string
		var size int64
		if err := rows.Scan(&hash, &size); err != nil {
			return GCResult{}, fmt.Errorf("cas: gc scan: %w", err)
		}
		hashes = append(hashes, hash)
		totalFreed += size
	}
	if err := rows.Err(); err != nil {
		return GCResult{}, fmt.Errorf("cas: gc rows: %w", err)
	}

	if len(hashes) == 0 {
		return GCResult{}, tx.Commit()
	}

	placeholders := make([]string, len(hashes))
	args := make([]interface{}, len(hashes))
	for i, h := range hashes {
		placeholders[i] = "?"
		args[i] = h
	}
	_, err = tx.ExecContext(ctx,
		`DELETE FROM content WHERE hash IN (`+strings.Join(placeholders, ",")+`)`,
		args...,
	)
	if err != nil {
		return GCResult{}, fmt.Errorf("cas: gc delete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return GCResult{}, fmt.Errorf("cas: gc commit: %w", err)
	}

	return GCResult{Removed: len(hashes), Freed: totalFreed}, nil
}

func (s *sqliteStore) Export(ctx context.Context, refs []Ref, w io.Writer) error {
	tw := tar.NewWriter(w)

	for _, ref := range refs {
		data, meta, err := s.Get(ctx, ref)
		if err != nil {
			tw.Close()
			return fmt.Errorf("cas: export get %s: %w", ref, err)
		}

		if err := tw.WriteHeader(&tar.Header{
			Name: string(ref) + ".blob",
			Size: int64(len(data)),
			Mode: 0644,
		}); err != nil {
			tw.Close()
			return fmt.Errorf("cas: export header: %w", err)
		}
		if _, err := tw.Write(data); err != nil {
			tw.Close()
			return fmt.Errorf("cas: export write: %w", err)
		}

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			tw.Close()
			return fmt.Errorf("cas: export marshal meta: %w", err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: string(ref) + ".meta.json",
			Size: int64(len(metaJSON)),
			Mode: 0644,
		}); err != nil {
			tw.Close()
			return fmt.Errorf("cas: export meta header: %w", err)
		}
		if _, err := tw.Write(metaJSON); err != nil {
			tw.Close()
			return fmt.Errorf("cas: export meta write: %w", err)
		}
	}
	// Explicitly check Close error instead of deferring. (#18)
	return tw.Close()
}

func (s *sqliteStore) Import(ctx context.Context, r io.Reader) ([]Ref, error) {
	tr := tar.NewReader(r)
	blobs := make(map[string][]byte)
	metas := make(map[string]ContentMeta)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cas: import read: %w", err)
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("cas: import read data: %w", err)
		}

		name := hdr.Name
		if strings.HasSuffix(name, ".meta.json") {
			hash := strings.TrimSuffix(name, ".meta.json")
			var meta ContentMeta
			if err := json.Unmarshal(data, &meta); err != nil {
				return nil, fmt.Errorf("cas: import unmarshal meta: %w", err)
			}
			metas[hash] = meta
		} else if strings.HasSuffix(name, ".blob") {
			hash := strings.TrimSuffix(name, ".blob")
			blobs[hash] = data
		}
	}

	var refs []Ref
	for expectedHash, blob := range blobs {
		meta := metas[expectedHash]
		ref, err := s.Put(ctx, blob, meta)
		if err != nil {
			return nil, fmt.Errorf("cas: import put: %w", err)
		}
		// Verify hash integrity: recomputed hash must match the tar filename. (#6)
		if string(ref) != expectedHash {
			return nil, fmt.Errorf("cas: import hash mismatch: expected %s, got %s", expectedHash, ref)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}
