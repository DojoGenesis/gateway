package cas

import (
	"archive/tar"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/runtime/d1client"
)

// D1Config holds connection parameters for the D1-backed CAS store.
type D1Config struct {
	// AccountID is the Cloudflare account identifier.
	AccountID string

	// DatabaseID is the D1 database ID (UUID).
	DatabaseID string

	// APIToken is the Cloudflare API token with D1 read/write permissions.
	// Recommended: use CLOUDFLARE_D1_TOKEN environment variable.
	APIToken string

	// BaseURL overrides the Cloudflare API base URL (used in tests).
	BaseURL string
}

// d1Store is the D1-backed CAS Store.
//
// Binary content is stored as base64-encoded text in the BLOB-affinity `data`
// column. D1's REST API transmits all parameters as JSON; base64 avoids the
// awkward byte-array encoding that D1 uses for raw BLOBs.
type d1Store struct {
	client *d1client.Client
}

// NewD1Store creates a D1-backed Store. The remote `content` and `tags` tables
// must already exist (matching the schema in runtime/cas/store.go createTables).
func NewD1Store(cfg D1Config) (Store, error) {
	if cfg.AccountID == "" || cfg.DatabaseID == "" || cfg.APIToken == "" {
		return nil, errors.New("cas/d1: AccountID, DatabaseID, and APIToken are required")
	}
	return &d1Store{
		client: d1client.New(d1client.Config{
			AccountID:  cfg.AccountID,
			DatabaseID: cfg.DatabaseID,
			APIToken:   cfg.APIToken,
			BaseURL:    cfg.BaseURL,
		}),
	}, nil
}

// ---------------------------------------------------------------------------
// Store interface implementation
// ---------------------------------------------------------------------------

func (s *d1Store) Put(ctx context.Context, content []byte, meta ContentMeta) (Ref, error) {
	ref := computeHash(content)
	meta.Size = int64(len(content))
	if meta.CreatedAt.IsZero() {
		meta.CreatedAt = time.Now().UTC()
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("cas/d1: marshal meta: %w", err)
	}

	// Encode binary content as base64; D1 stores it as text in the BLOB column.
	encoded := base64.StdEncoding.EncodeToString(content)

	_, err = s.client.Exec(ctx,
		`INSERT INTO content (hash, data, meta, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(hash) DO UPDATE SET meta = excluded.meta, created_at = excluded.created_at`,
		string(ref), encoded, string(metaJSON), meta.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return "", fmt.Errorf("cas/d1: put: %w", err)
	}
	return ref, nil
}

func (s *d1Store) Get(ctx context.Context, ref Ref) ([]byte, ContentMeta, error) {
	rows, err := s.client.Query(ctx,
		`SELECT data, meta FROM content WHERE hash = ?`, string(ref),
	)
	if err != nil {
		return nil, ContentMeta{}, fmt.Errorf("cas/d1: get: %w", err)
	}
	if len(rows) == 0 {
		return nil, ContentMeta{}, ErrNotFound
	}

	encoded := d1client.String(rows[0]["data"])
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ContentMeta{}, fmt.Errorf("cas/d1: decode data: %w", err)
	}

	var meta ContentMeta
	if err := json.Unmarshal([]byte(d1client.String(rows[0]["meta"])), &meta); err != nil {
		return nil, ContentMeta{}, fmt.Errorf("cas/d1: unmarshal meta: %w", err)
	}
	return content, meta, nil
}

func (s *d1Store) Has(ctx context.Context, ref Ref) (bool, error) {
	rows, err := s.client.Query(ctx,
		`SELECT COUNT(*) AS n FROM content WHERE hash = ?`, string(ref),
	)
	if err != nil {
		return false, fmt.Errorf("cas/d1: has: %w", err)
	}
	if len(rows) == 0 {
		return false, nil
	}
	return d1client.Int64(rows[0]["n"]) > 0, nil
}

func (s *d1Store) Tag(ctx context.Context, name string, version string, ref Ref) error {
	exists, err := s.Has(ctx, ref)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("cas/d1: tag: ref %s not found", ref)
	}

	_, err = s.client.Exec(ctx,
		`INSERT INTO tags (name, version, ref, created_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(name, version) DO UPDATE SET ref = excluded.ref, created_at = excluded.created_at`,
		name, version, string(ref), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("cas/d1: tag: %w", err)
	}
	return nil
}

func (s *d1Store) Untag(ctx context.Context, name string, version string) error {
	n, err := s.client.Exec(ctx,
		`DELETE FROM tags WHERE name = ? AND version = ?`, name, version,
	)
	if err != nil {
		return fmt.Errorf("cas/d1: untag: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *d1Store) Resolve(ctx context.Context, name string, version string) (Ref, error) {
	rows, err := s.client.Query(ctx,
		`SELECT ref FROM tags WHERE name = ? AND version = ?`, name, version,
	)
	if err != nil {
		return "", fmt.Errorf("cas/d1: resolve: %w", err)
	}
	if len(rows) == 0 {
		return "", ErrNotFound
	}
	return Ref(d1client.String(rows[0]["ref"])), nil
}

func (s *d1Store) List(ctx context.Context, prefix string) ([]TagEntry, error) {
	rows, err := s.client.Query(ctx,
		`SELECT t.name, t.version, t.ref, c.meta
		 FROM tags t JOIN content c ON t.ref = c.hash
		 WHERE t.name LIKE ?
		 ORDER BY t.name, t.version`,
		prefix+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("cas/d1: list: %w", err)
	}

	entries := make([]TagEntry, 0, len(rows))
	for _, row := range rows {
		var e TagEntry
		e.Name = d1client.String(row["name"])
		e.Version = d1client.String(row["version"])
		e.Ref = Ref(d1client.String(row["ref"]))

		if err := json.Unmarshal([]byte(d1client.String(row["meta"])), &e.Meta); err != nil {
			return nil, fmt.Errorf("cas/d1: list unmarshal: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// GC removes content not referenced by any tag.
//
// Note: D1's REST API is stateless; this operation is not atomic. A small race
// window exists between the SELECT and DELETE. Acceptable for the current scale.
func (s *d1Store) GC(ctx context.Context) (GCResult, error) {
	// Find unreferenced content.
	rows, err := s.client.Query(ctx,
		`SELECT c.hash, LENGTH(c.data) AS sz
		 FROM content c
		 LEFT JOIN tags t ON c.hash = t.ref
		 WHERE t.ref IS NULL`,
	)
	if err != nil {
		return GCResult{}, fmt.Errorf("cas/d1: gc query: %w", err)
	}
	if len(rows) == 0 {
		return GCResult{}, nil
	}

	var hashes []string
	var freed int64
	for _, row := range rows {
		hashes = append(hashes, d1client.String(row["hash"]))
		// sz is the byte-length of the base64 string; approximate freed bytes.
		freed += d1client.Int64(row["sz"])
	}

	placeholders := strings.Repeat("?,", len(hashes))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(hashes))
	for i, h := range hashes {
		args[i] = h
	}

	_, err = s.client.Exec(ctx,
		`DELETE FROM content WHERE hash IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return GCResult{}, fmt.Errorf("cas/d1: gc delete: %w", err)
	}

	return GCResult{Removed: len(hashes), Freed: freed}, nil
}

// Export writes content to a tar archive. Identical logic to sqliteStore.Export.
func (s *d1Store) Export(ctx context.Context, refs []Ref, w io.Writer) error {
	tw := tar.NewWriter(w)

	for _, ref := range refs {
		data, meta, err := s.Get(ctx, ref)
		if err != nil {
			tw.Close()
			return fmt.Errorf("cas/d1: export get %s: %w", ref, err)
		}

		if err := tw.WriteHeader(&tar.Header{
			Name: string(ref) + ".blob",
			Size: int64(len(data)),
			Mode: 0644,
		}); err != nil {
			tw.Close()
			return fmt.Errorf("cas/d1: export header: %w", err)
		}
		if _, err := tw.Write(data); err != nil {
			tw.Close()
			return fmt.Errorf("cas/d1: export write: %w", err)
		}

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			tw.Close()
			return fmt.Errorf("cas/d1: export meta marshal: %w", err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name: string(ref) + ".meta.json",
			Size: int64(len(metaJSON)),
			Mode: 0644,
		}); err != nil {
			tw.Close()
			return fmt.Errorf("cas/d1: export meta header: %w", err)
		}
		if _, err := tw.Write(metaJSON); err != nil {
			tw.Close()
			return fmt.Errorf("cas/d1: export meta write: %w", err)
		}
	}
	return tw.Close()
}

// Import reads content from a tar archive. Identical logic to sqliteStore.Import.
func (s *d1Store) Import(ctx context.Context, r io.Reader) ([]Ref, error) {
	tr := tar.NewReader(r)
	blobs := make(map[string][]byte)
	metas := make(map[string]ContentMeta)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cas/d1: import read: %w", err)
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("cas/d1: import read data: %w", err)
		}

		name := hdr.Name
		switch {
		case strings.HasSuffix(name, ".meta.json"):
			hash := strings.TrimSuffix(name, ".meta.json")
			var meta ContentMeta
			if err := json.Unmarshal(data, &meta); err != nil {
				return nil, fmt.Errorf("cas/d1: import unmarshal meta: %w", err)
			}
			metas[hash] = meta
		case strings.HasSuffix(name, ".blob"):
			hash := strings.TrimSuffix(name, ".blob")
			blobs[hash] = data
		}
	}

	var refs []Ref
	for expectedHash, blob := range blobs {
		meta := metas[expectedHash]
		ref, err := s.Put(ctx, blob, meta)
		if err != nil {
			return nil, fmt.Errorf("cas/d1: import put: %w", err)
		}
		if string(ref) != expectedHash {
			return nil, fmt.Errorf("cas/d1: import hash mismatch: expected %s, got %s", expectedHash, ref)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// Close is a no-op; D1 uses stateless HTTP with no persistent connection.
func (s *d1Store) Close() error { return nil }
