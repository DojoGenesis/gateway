package skill

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// validateTar checks that data is a valid tar archive by reading the first header.
func validateTar(data []byte) error {
	tr := tar.NewReader(bytes.NewReader(data))
	_, err := tr.Next()
	if err != nil {
		return fmt.Errorf("invalid tar archive: %w", err)
	}
	return nil
}

// tagPrefix is the namespace for skill tags in CAS.
const tagPrefix = "skill/"

// SkillStore manages skills in CAS with OCI-compatible tagging.
//
// Each installed skill occupies two CAS blobs:
//   - A config blob (JSON manifest) tagged as "skill/{name}:config" at version V
//   - A content blob (tar archive) tagged as "skill/{name}:content" at version V
//
// This two-blob layout mirrors the OCI image manifest pattern: a config layer
// for metadata and one or more content layers for data.
type SkillStore struct {
	cas cas.Store
}

// NewSkillStore creates a SkillStore backed by the given CAS store.
func NewSkillStore(casStore cas.Store) *SkillStore {
	return &SkillStore{cas: casStore}
}

// configTagName returns the CAS tag name for a skill's config blob.
func configTagName(name string) string {
	return tagPrefix + name + ":config"
}

// contentTagName returns the CAS tag name for a skill's content blob.
func contentTagName(name string) string {
	return tagPrefix + name + ":content"
}

// Install stores a packaged skill into CAS.
//
// It validates that contentTar is a valid tar archive and that configBlob
// deserializes to a manifest matching the provided one, then stores both
// blobs and tags them.
//
// Operation order for rollback safety:
//   1. Validate inputs (tar, config blob match)
//   2. Store both blobs first (idempotent, safe to retry)
//   3. Tags are applied after both blobs are stored
//   4. If tagging fails, orphaned blobs are cleaned up by CAS GC
func (s *SkillStore) Install(ctx context.Context, manifest SkillManifest, configBlob, contentTar []byte) error {
	if manifest.Name == "" {
		return fmt.Errorf("skill: install: manifest name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("skill: install: manifest version is required")
	}

	// E30: Validate contentTar is a valid tar archive.
	if err := validateTar(contentTar); err != nil {
		return fmt.Errorf("skill: install: %w", err)
	}

	// E31: Validate configBlob matches manifest name and version.
	var blobManifest SkillManifest
	if err := json.Unmarshal(configBlob, &blobManifest); err != nil {
		return fmt.Errorf("skill: install: invalid config blob: %w", err)
	}
	if blobManifest.Name != manifest.Name || blobManifest.Version != manifest.Version {
		return fmt.Errorf("skill: install: config blob name/version mismatch: blob=%s@%s, manifest=%s@%s",
			blobManifest.Name, blobManifest.Version, manifest.Name, manifest.Version)
	}

	// E32: Store both blobs first (idempotent, safe to retry).
	// Tags are applied after both blobs are stored.
	// If tagging fails, orphaned blobs are cleaned up by CAS GC.

	// Put config blob.
	configRef, err := s.cas.Put(ctx, configBlob, cas.ContentMeta{
		Type:      cas.ContentSkill,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"mediaType": ConfigMediaType,
			"skill":     manifest.Name,
			"version":   manifest.Version,
		},
	})
	if err != nil {
		return fmt.Errorf("skill: install: put config: %w", err)
	}

	// Put content blob.
	contentRef, err := s.cas.Put(ctx, contentTar, cas.ContentMeta{
		Type:      cas.ContentSkill,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"mediaType": ContentMediaType,
			"skill":     manifest.Name,
			"version":   manifest.Version,
		},
	})
	if err != nil {
		return fmt.Errorf("skill: install: put content: %w", err)
	}

	// Tag config blob (only after both Puts succeed).
	if err := s.cas.Tag(ctx, configTagName(manifest.Name), manifest.Version, configRef); err != nil {
		return fmt.Errorf("skill: install: tag config: %w", err)
	}

	// Tag content blob.
	if err := s.cas.Tag(ctx, contentTagName(manifest.Name), manifest.Version, contentRef); err != nil {
		return fmt.Errorf("skill: install: tag content: %w", err)
	}

	return nil
}

// List returns all installed skills by scanning config tags.
//
// It lists all tags with the "skill/" prefix and deserializes the config blob
// of each unique skill into a SkillManifest.
func (s *SkillStore) List(ctx context.Context) ([]SkillManifest, error) {
	entries, err := s.cas.List(ctx, tagPrefix)
	if err != nil {
		return nil, fmt.Errorf("skill: list: %w", err)
	}

	var manifests []SkillManifest
	seen := make(map[string]bool)

	for _, entry := range entries {
		// Only process config tags (skip content tags).
		if !isConfigTag(entry.Name) {
			continue
		}

		// Deduplicate by name+version.
		key := entry.Name + "@" + entry.Version
		if seen[key] {
			continue
		}
		seen[key] = true

		// Fetch and deserialize the config blob.
		data, _, err := s.cas.Get(ctx, entry.Ref)
		if err != nil {
			return nil, fmt.Errorf("skill: list: get config for %s: %w", entry.Name, err)
		}

		var m SkillManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("skill: list: unmarshal config for %s: %w", entry.Name, err)
		}
		manifests = append(manifests, m)
	}

	return manifests, nil
}

// Get retrieves an installed skill's manifest by name and version.
func (s *SkillStore) Get(ctx context.Context, name, version string) (*SkillManifest, error) {
	ref, err := s.cas.Resolve(ctx, configTagName(name), version)
	if err != nil {
		return nil, fmt.Errorf("skill: get %s@%s: %w", name, version, err)
	}

	data, _, err := s.cas.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("skill: get %s@%s data: %w", name, version, err)
	}

	var m SkillManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("skill: get %s@%s unmarshal: %w", name, version, err)
	}

	return &m, nil
}

// GetContent retrieves the skill's content tar archive by name and version.
func (s *SkillStore) GetContent(ctx context.Context, name, version string) ([]byte, error) {
	ref, err := s.cas.Resolve(ctx, contentTagName(name), version)
	if err != nil {
		return nil, fmt.Errorf("skill: get content %s@%s: %w", name, version, err)
	}

	data, _, err := s.cas.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("skill: get content %s@%s data: %w", name, version, err)
	}

	return data, nil
}

// isConfigTag checks if a tag name is a config tag (ends with ":config").
func isConfigTag(name string) bool {
	return strings.HasSuffix(name, ":config")
}
