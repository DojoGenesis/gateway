package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RemoteFetcher pulls skills from OCI registries using the OCI Distribution
// API (v2). It avoids a hard dependency on oras-go by implementing the
// minimal fetch flow directly: resolve manifest → download config layer →
// download content layer.
//
// The flow:
//  1. GET /v2/{path}/manifests/{tag} → OCI manifest JSON
//  2. Parse config descriptor and content layer descriptor
//  3. GET /v2/{path}/blobs/{digest} for each layer
//  4. Return manifest + config + content blobs
type RemoteFetcher struct {
	store  *SkillStore
	client *http.Client
}

// NewRemoteFetcher creates a fetcher backed by the given store.
func NewRemoteFetcher(store *SkillStore) *RemoteFetcher {
	return &RemoteFetcher{
		store:  store,
		client: &http.Client{},
	}
}

// ociManifest is a minimal representation of an OCI Image Manifest.
type ociManifest struct {
	SchemaVersion int             `json:"schemaVersion"`
	MediaType     string          `json:"mediaType"`
	ArtifactType  string          `json:"artifactType,omitempty"`
	Config        ociDescriptor   `json:"config"`
	Layers        []ociDescriptor `json:"layers"`
}

// ociDescriptor is a content descriptor (digest + size + media type).
type ociDescriptor struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

// Fetch pulls a skill from an OCI registry.
//
// It resolves the manifest by tag, downloads the config blob and content
// layer, and returns them for installation into the local CAS.
func (f *RemoteFetcher) Fetch(ctx context.Context, ref ResolvedRef) (*SkillManifest, []byte, []byte, error) {
	baseURL := fmt.Sprintf("https://%s/v2/%s", ref.Registry, ref.Path)

	// Step 1: Resolve manifest.
	manifestURL := fmt.Sprintf("%s/manifests/%s", baseURL, ref.Tag)
	manifestData, err := f.fetchJSON(ctx, manifestURL, "application/vnd.oci.image.manifest.v1+json")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetch manifest: %w", err)
	}

	var manifest ociManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, nil, nil, fmt.Errorf("parse manifest: %w", err)
	}

	// Step 2: Download config blob.
	if manifest.Config.Digest == "" {
		return nil, nil, nil, fmt.Errorf("manifest has no config descriptor")
	}
	configURL := fmt.Sprintf("%s/blobs/%s", baseURL, manifest.Config.Digest)
	configBlob, err := f.fetchBlob(ctx, configURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetch config: %w", err)
	}

	var skillManifest SkillManifest
	if err := json.Unmarshal(configBlob, &skillManifest); err != nil {
		return nil, nil, nil, fmt.Errorf("parse config: %w", err)
	}

	// Step 3: Download content layer (first layer with content media type).
	var contentTar []byte
	for _, layer := range manifest.Layers {
		if layer.MediaType == ContentMediaType || strings.HasSuffix(layer.MediaType, "+tar") {
			layerURL := fmt.Sprintf("%s/blobs/%s", baseURL, layer.Digest)
			contentTar, err = f.fetchBlob(ctx, layerURL)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("fetch content layer: %w", err)
			}
			break
		}
	}
	if contentTar == nil {
		return nil, nil, nil, fmt.Errorf("no content layer found in manifest")
	}

	return &skillManifest, configBlob, contentTar, nil
}

// fetchJSON performs a GET with the given Accept header and returns the body.
func (f *RemoteFetcher) fetchJSON(ctx context.Context, url, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// fetchBlob downloads a blob by URL and returns the raw bytes.
func (f *RemoteFetcher) fetchBlob(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
