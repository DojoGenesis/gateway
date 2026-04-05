package skill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultRekorBaseURL = "https://rekor.sigstore.dev"

// RekorLogEntry is a decoded Rekor transparency log entry.
type RekorLogEntry struct {
	// UUID is the entry identifier returned by the Rekor index/retrieve API.
	UUID string

	// LogIndex is the position of the entry in the log.
	LogIndex int64

	// IntegratedTime is when the entry was incorporated into the log.
	IntegratedTime time.Time

	// EntryURL is the direct link to the entry in the Rekor search UI.
	EntryURL string
}

// RekorClient queries the Sigstore Rekor transparency log REST API.
//
// It is intentionally minimal — only the two endpoints used by the Gateway
// are implemented: hash lookup and entry retrieval. No sigstore-go dependency;
// the same CLI-shell-out philosophy used in verify.go.
type RekorClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRekorClient creates a RekorClient pointing at baseURL.
// Pass an empty string to use the Sigstore public log (rekor.sigstore.dev).
func NewRekorClient(baseURL string) *RekorClient {
	if baseURL == "" {
		baseURL = defaultRekorBaseURL
	}
	return &RekorClient{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// LookupByHash queries Rekor for transparency log entries by artifact SHA-256 hash.
//
// hash must be in the form "sha256:<hex>" (e.g. "sha256:abc123…").
// Returns log entry UUIDs — typically zero or one entry for a given artifact.
func (c *RekorClient) LookupByHash(ctx context.Context, hash string) ([]string, error) {
	body, err := json.Marshal(map[string]string{"hash": hash})
	if err != nil {
		return nil, fmt.Errorf("rekor: marshal lookup request: %w", err)
	}

	url := c.baseURL + "/api/v1/index/retrieve"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rekor: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rekor: lookup: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rekor: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rekor: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var uuids []string
	if err := json.Unmarshal(raw, &uuids); err != nil {
		return nil, fmt.Errorf("rekor: decode lookup response: %w", err)
	}
	return uuids, nil
}

// GetEntry retrieves a specific Rekor log entry by UUID.
//
// Rekor returns a map keyed by UUID; GetEntry extracts and decodes the single
// entry.
func (c *RekorClient) GetEntry(ctx context.Context, uuid string) (*RekorLogEntry, error) {
	url := c.baseURL + "/api/v1/log/entries/" + uuid
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("rekor: build entry request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rekor: get entry: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("rekor: read entry response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rekor: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	// Rekor returns { "<uuid>": { "body": "...", "integratedTime": N, "logIndex": N, ... } }
	var outer map[string]struct {
		Body           string `json:"body"`
		IntegratedTime int64  `json:"integratedTime"`
		LogIndex       int64  `json:"logIndex"`
		LogID          string `json:"logID"`
	}
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("rekor: decode entry: %w", err)
	}
	if len(outer) == 0 {
		return nil, fmt.Errorf("rekor: entry %s not found", uuid)
	}

	// The map has exactly one key — the UUID itself.
	var e RekorLogEntry
	for k, v := range outer {
		e.UUID = k
		e.LogIndex = v.LogIndex
		e.EntryURL = fmt.Sprintf("%s/?logIndex=%d", rekorSearchBaseURL, v.LogIndex)
		if v.IntegratedTime > 0 {
			e.IntegratedTime = time.Unix(v.IntegratedTime, 0).UTC()
		}
		break
	}
	return &e, nil
}

// MonitorCASArtifact checks whether a CAS artifact is present in the Rekor
// transparency log by its SHA-256 hash.
//
// hexHash is the hex-encoded SHA-256 without any "sha256:" prefix — matching
// the CAS Ref type (see runtime/cas/ref.go). Returns nil when the artifact
// has no Rekor entry (unsigned / community tier).
func MonitorCASArtifact(ctx context.Context, client *RekorClient, hexHash string) (*RekorLogEntry, error) {
	uuids, err := client.LookupByHash(ctx, "sha256:"+hexHash)
	if err != nil {
		return nil, fmt.Errorf("rekor monitor: lookup %s: %w", hexHash, err)
	}
	if len(uuids) == 0 {
		return nil, nil // not in log — community tier
	}

	entry, err := client.GetEntry(ctx, uuids[0])
	if err != nil {
		return nil, fmt.Errorf("rekor monitor: get entry: %w", err)
	}
	return entry, nil
}
