package skill

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/runtime/cas"
)

// CASStorage adapts the CAS Store to an OCI-compatible content storage interface.
//
// OCI distribution uses content-addressed storage keyed by digest strings in
// the format "sha256:<hex>". CAS uses bare SHA-256 hex strings as Ref values.
// CASStorage bridges between these two representations, making CAS usable as
// the storage backend for OCI artifact push/pull operations.
//
// This adapter implements a subset of the oras-go Storage interface without
// importing the oras-go library, keeping the dependency graph clean.
type CASStorage struct {
	store cas.Store
}

// NewCASStorage wraps a CAS Store to provide OCI-compatible storage operations.
func NewCASStorage(store cas.Store) *CASStorage {
	return &CASStorage{store: store}
}

// Exists checks if content with the given OCI digest exists in CAS.
//
// The digest must be in the format "sha256:<hex>". It is converted to a bare
// CAS Ref for the lookup.
func (s *CASStorage) Exists(ctx context.Context, digest string) (bool, error) {
	ref, err := digestToRef(digest)
	if err != nil {
		return false, err
	}
	return s.store.Has(ctx, ref)
}

// Fetch retrieves content by OCI digest from CAS.
//
// The digest format "sha256:<hex>" is converted to a CAS Ref for retrieval.
// Returns the raw content bytes.
func (s *CASStorage) Fetch(ctx context.Context, digest string) ([]byte, error) {
	ref, err := digestToRef(digest)
	if err != nil {
		return nil, err
	}

	data, _, err := s.store.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("oras: fetch %s: %w", digest, err)
	}
	return data, nil
}

// Push stores content in CAS and returns its OCI digest.
//
// The content is stored with the given media type as a label. The returned
// digest is in OCI format "sha256:<hex>".
func (s *CASStorage) Push(ctx context.Context, data []byte, mediaType string) (string, error) {
	meta := cas.ContentMeta{
		Type: cas.ContentSkill,
		Labels: map[string]string{
			"mediaType": mediaType,
		},
	}

	ref, err := s.store.Put(ctx, data, meta)
	if err != nil {
		return "", fmt.Errorf("oras: push: %w", err)
	}

	return refToDigest(ref), nil
}

// Tag assigns a name and version tag to a content digest.
//
// This maps directly to CAS tagging, converting the OCI digest to a CAS Ref
// before delegating.
func (s *CASStorage) Tag(ctx context.Context, name, version, digest string) error {
	ref, err := digestToRef(digest)
	if err != nil {
		return err
	}
	return s.store.Tag(ctx, name, version, ref)
}

// Resolve looks up a tag by name and version, returning the OCI digest.
func (s *CASStorage) Resolve(ctx context.Context, name, version string) (string, error) {
	ref, err := s.store.Resolve(ctx, name, version)
	if err != nil {
		return "", fmt.Errorf("oras: resolve %s@%s: %w", name, version, err)
	}
	return refToDigest(ref), nil
}

// Digest computes the OCI digest for the given data without storing it.
//
// Returns the digest in "sha256:<hex>" format.
func Digest(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

// digestToRef converts an OCI digest ("sha256:<hex>") to a CAS Ref.
func digestToRef(digest string) (cas.Ref, error) {
	if !strings.HasPrefix(digest, "sha256:") {
		return "", fmt.Errorf("oras: unsupported digest algorithm (expected sha256:): %s", digest)
	}
	hexStr := strings.TrimPrefix(digest, "sha256:")
	if len(hexStr) != 64 {
		return "", fmt.Errorf("oras: invalid digest length: %s", digest)
	}
	return cas.Ref(hexStr), nil
}

// refToDigest converts a CAS Ref (bare hex) to an OCI digest ("sha256:<hex>").
func refToDigest(ref cas.Ref) string {
	return "sha256:" + string(ref)
}
