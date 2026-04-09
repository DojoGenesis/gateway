package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/DojoGenesis/gateway/runtime/cas"
)

// Yank marks a skill version as yanked in CAS. The content tar is NOT deleted
// — only the config blob is updated with Yanked=true and YankReason. This
// implements the "yank never delete" policy from ADR-020.
//
// The updated config blob gets a new CAS ref (content-addressed); the config
// tag is re-pointed to the new ref. The old config blob remains in CAS until
// GC, preserving auditability.
func (s *SkillStore) Yank(ctx context.Context, name, version, reason string) error {
	if name == "" {
		return fmt.Errorf("skill: yank: name is required")
	}
	if version == "" {
		return fmt.Errorf("skill: yank: version is required")
	}
	if reason == "" {
		return fmt.Errorf("skill: yank: reason is required")
	}

	// Read the current config blob.
	ref, err := s.cas.Resolve(ctx, configTagName(name), version)
	if err != nil {
		return fmt.Errorf("skill: yank %s@%s: resolve: %w", name, version, err)
	}

	data, _, err := s.cas.Get(ctx, ref)
	if err != nil {
		return fmt.Errorf("skill: yank %s@%s: get config: %w", name, version, err)
	}

	// Unmarshal as PluginManifest to access the Yanked/YankReason fields.
	var pm PluginManifest
	if err := json.Unmarshal(data, &pm); err != nil {
		return fmt.Errorf("skill: yank %s@%s: unmarshal config: %w", name, version, err)
	}

	if pm.Yanked {
		return fmt.Errorf("skill: %s@%s is already yanked", name, version)
	}

	pm.Yanked = true
	pm.YankReason = reason

	// Write the updated config blob back to CAS.
	updated, err := json.Marshal(pm)
	if err != nil {
		return fmt.Errorf("skill: yank %s@%s: marshal config: %w", name, version, err)
	}

	newRef, err := s.cas.Put(ctx, updated, cas.ContentMeta{
		Type:      cas.ContentSkill,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"mediaType": ConfigMediaType,
			"skill":     name,
			"version":   version,
			"yanked":    "true",
		},
	})
	if err != nil {
		return fmt.Errorf("skill: yank %s@%s: put config: %w", name, version, err)
	}

	// Re-point the config tag to the new blob.
	if err := s.cas.Tag(ctx, configTagName(name), version, newRef); err != nil {
		return fmt.Errorf("skill: yank %s@%s: tag config: %w", name, version, err)
	}

	return nil
}

// UnYank reverses a yank, restoring the skill to installable state. The config
// blob is updated with Yanked=false and YankReason cleared.
func (s *SkillStore) UnYank(ctx context.Context, name, version string) error {
	if name == "" {
		return fmt.Errorf("skill: unyank: name is required")
	}
	if version == "" {
		return fmt.Errorf("skill: unyank: version is required")
	}

	// Read the current config blob.
	ref, err := s.cas.Resolve(ctx, configTagName(name), version)
	if err != nil {
		return fmt.Errorf("skill: unyank %s@%s: resolve: %w", name, version, err)
	}

	data, _, err := s.cas.Get(ctx, ref)
	if err != nil {
		return fmt.Errorf("skill: unyank %s@%s: get config: %w", name, version, err)
	}

	var pm PluginManifest
	if err := json.Unmarshal(data, &pm); err != nil {
		return fmt.Errorf("skill: unyank %s@%s: unmarshal config: %w", name, version, err)
	}

	if !pm.Yanked {
		return fmt.Errorf("skill: %s@%s is not yanked", name, version)
	}

	pm.Yanked = false
	pm.YankReason = ""

	updated, err := json.Marshal(pm)
	if err != nil {
		return fmt.Errorf("skill: unyank %s@%s: marshal config: %w", name, version, err)
	}

	newRef, err := s.cas.Put(ctx, updated, cas.ContentMeta{
		Type:      cas.ContentSkill,
		CreatedAt: time.Now().UTC(),
		Labels: map[string]string{
			"mediaType": ConfigMediaType,
			"skill":     name,
			"version":   version,
		},
	})
	if err != nil {
		return fmt.Errorf("skill: unyank %s@%s: put config: %w", name, version, err)
	}

	if err := s.cas.Tag(ctx, configTagName(name), version, newRef); err != nil {
		return fmt.Errorf("skill: unyank %s@%s: tag config: %w", name, version, err)
	}

	return nil
}
