package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"archive/tar"
	"bytes"
	"io"
	"log/slog"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	pkgskill "github.com/DojoGenesis/gateway/pkg/skill"
)

// SkillRegistry manages skill registration, lookup, and metadata.
type SkillRegistry interface {
	// RegisterSkill adds a skill to the registry.
	// Returns error if skill name already exists or metadata is invalid.
	RegisterSkill(ctx context.Context, skill *SkillDefinition) error

	// GetSkill retrieves a skill by exact name.
	GetSkill(ctx context.Context, name string) (*SkillDefinition, error)

	// ListSkills returns all registered skills.
	ListSkills(ctx context.Context) ([]*SkillDefinition, error)

	// ListByPlugin returns skills for a specific plugin.
	ListByPlugin(ctx context.Context, pluginName string) ([]*SkillDefinition, error)

	// ListByTier returns skills filtered by portability tier.
	ListByTier(ctx context.Context, tier int) ([]*SkillDefinition, error)

	// ListByAgent returns skills available to a specific agent.
	ListByAgent(ctx context.Context, agentName string) ([]*SkillDefinition, error)

	// LoadFromDirectory scans a directory for SKILL.md files and registers them.
	LoadFromDirectory(ctx context.Context, dirPath string, pluginName string) error

	// LoadFromManifest loads skills from a plugin manifest file.
	LoadFromManifest(ctx context.Context, manifestPath string) error
}

// InMemorySkillRegistry is an in-memory implementation of SkillRegistry
type InMemorySkillRegistry struct {
	skills map[string]*SkillDefinition
	mu     sync.RWMutex
}

// NewInMemorySkillRegistry creates a new in-memory skill registry
func NewInMemorySkillRegistry() *InMemorySkillRegistry {
	return &InMemorySkillRegistry{
		skills: make(map[string]*SkillDefinition),
	}
}

// RegisterSkill adds a skill to the registry
func (r *InMemorySkillRegistry) RegisterSkill(ctx context.Context, skill *SkillDefinition) error {
	if skill == nil {
		return fmt.Errorf("skill definition cannot be nil")
	}

	// Validate skill metadata
	if err := skill.IsValid(); err != nil {
		return fmt.Errorf("skill validation failed: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	if _, exists := r.skills[skill.Name]; exists {
		return fmt.Errorf("%w: %s", ErrSkillAlreadyExists, skill.Name)
	}

	// Assign UUID if not already set
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}

	// Set loaded timestamp
	skill.LoadedAt = time.Now()

	r.skills[skill.Name] = skill
	return nil
}

// GetSkill retrieves a skill by name
func (r *InMemorySkillRegistry) GetSkill(ctx context.Context, name string) (*SkillDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrSkillNotFound, name)
	}
	return skill, nil
}

// ListSkills returns all registered skills
func (r *InMemorySkillRegistry) ListSkills(ctx context.Context) ([]*SkillDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SkillDefinition, 0, len(r.skills))
	for _, skill := range r.skills {
		result = append(result, skill)
	}
	return result, nil
}

// ListByPlugin returns skills for a specific plugin
func (r *InMemorySkillRegistry) ListByPlugin(ctx context.Context, pluginName string) ([]*SkillDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SkillDefinition, 0)
	for _, skill := range r.skills {
		if skill.PluginName == pluginName {
			result = append(result, skill)
		}
	}
	return result, nil
}

// ListByTier returns skills filtered by portability tier
func (r *InMemorySkillRegistry) ListByTier(ctx context.Context, tier int) ([]*SkillDefinition, error) {
	if tier < 1 || tier > 4 {
		return nil, ErrInvalidSkillTier
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SkillDefinition, 0)
	for _, skill := range r.skills {
		if skill.Tier == tier {
			result = append(result, skill)
		}
	}
	return result, nil
}

// ListByAgent returns skills available to a specific agent
func (r *InMemorySkillRegistry) ListByAgent(ctx context.Context, agentName string) ([]*SkillDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SkillDefinition, 0)
	for _, skill := range r.skills {
		// Skip hidden skills for agent queries
		if skill.Hidden {
			continue
		}

		// Check if agent is in the allowed list
		for _, agent := range skill.Agents {
			if agent == agentName {
				result = append(result, skill)
				break
			}
		}
	}
	return result, nil
}

// LoadFromDirectory scans a directory for SKILL.md files and registers them
func (r *InMemorySkillRegistry) LoadFromDirectory(ctx context.Context, dirPath string, pluginName string) error {
	var errors []error

	// Walk the directory tree looking for SKILL.md files
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process SKILL.md files
		if filepath.Base(path) != "SKILL.md" {
			return nil
		}

		// Parse and register the skill
		skill, parseErr := parseSkillFile(path, pluginName)
		if parseErr != nil {
			errors = append(errors, fmt.Errorf("failed to parse %s: %w", path, parseErr))
			return nil // Continue walking
		}

		// Register the skill
		regErr := r.RegisterSkill(ctx, skill)
		if regErr != nil {
			errors = append(errors, fmt.Errorf("failed to register skill from %s: %w", path, regErr))
			return nil // Continue walking
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("directory walk failed: %w", err)
	}

	// Return aggregated errors if any
	if len(errors) > 0 {
		return fmt.Errorf("loaded directory with %d errors: %v", len(errors), errors)
	}

	return nil
}

// LoadFromManifest loads skills from a plugin manifest file.
// Not yet implemented — use LoadFromDirectory instead.
func (r *InMemorySkillRegistry) LoadFromManifest(ctx context.Context, manifestPath string) error {
	return fmt.Errorf("manifest loading not implemented (use LoadFromDirectory instead)")
}

// parseSkillFile reads a SKILL.md file and extracts metadata + content
func parseSkillFile(filePath string, pluginName string) (*SkillDefinition, error) {
	// Read file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)

	// Extract YAML frontmatter (between --- markers)
	frontmatter, body, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// Parse YAML frontmatter
	var meta Metadata
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidYAMLFrontmatter, err)
	}

	// Handle both nested (spec-compliant) and flat (backward compat) formats
	var toolDeps []string
	var tier int
	var portable bool
	var agents []string
	var reqVersion string
	var pyScripts []string
	var shScripts []string
	var hidden bool
	var version, created, author string

	if meta.MetadataBlock != nil {
		// Nested format (per spec)
		toolDeps = meta.MetadataBlock.ToolDependencies
		tier = meta.MetadataBlock.Tier
		portable = meta.MetadataBlock.Portable
		agents = meta.MetadataBlock.Agents
		reqVersion = meta.MetadataBlock.RequiresVersion
		pyScripts = meta.MetadataBlock.PythonScripts
		shScripts = meta.MetadataBlock.ShellScripts
		hidden = meta.MetadataBlock.Hidden
		version = meta.MetadataBlock.Version
		created = meta.MetadataBlock.Created
		author = meta.MetadataBlock.Author
	} else {
		// Flat format (backward compat)
		toolDeps = meta.ToolDependencies
		tier = meta.Tier
		portable = meta.Portable
		agents = meta.Agents
		reqVersion = meta.RequiredVersion
		pyScripts = meta.PythonScripts
		shScripts = meta.ShellScripts
		hidden = meta.Hidden
		version = meta.Version
		created = meta.Created
		author = meta.Author
	}

	// Create SkillDefinition from metadata
	skill := &SkillDefinition{
		Name:             meta.Name,
		Description:      meta.Description,
		PluginName:       pluginName,
		FilePath:         filePath,
		Triggers:         meta.Triggers,
		ToolDependencies: toolDeps,
		Tier:             tier,
		Portable:         portable,
		Agents:           agents,
		RequiredVersion:  reqVersion,
		PythonScripts:    pyScripts,
		ShellScripts:     shScripts,
		Hidden:           hidden,
		Content:          body,
		ParsedAt:         time.Now(),
		Version:          version,
		Created:          created,
		Author:           author,
	}

	return skill, nil
}

// extractFrontmatter separates YAML frontmatter from markdown body
func extractFrontmatter(content string) (frontmatter string, body string, err error) {
	lines := strings.Split(content, "\n")

	// Find first ---
	startIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return "", "", ErrMissingFrontmatter
	}

	// Find second ---
	endIdx := -1
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		return "", "", ErrMissingFrontmatter
	}

	// Extract frontmatter (between the --- markers)
	frontmatter = strings.Join(lines[startIdx+1:endIdx], "\n")

	// Extract body (everything after the second ---)
	if endIdx+1 < len(lines) {
		body = strings.Join(lines[endIdx+1:], "\n")
	}

	return frontmatter, strings.TrimSpace(body), nil
}

// LoadFromCAS loads skills from a CAS-backed SkillStore into the registry.
// It reads each packaged skill's content tar, extracts the SKILL.md body,
// re-parses frontmatter for full field population, and registers the result.
// Skills that fail to parse or validate are logged and skipped.
func (r *InMemorySkillRegistry) LoadFromCAS(ctx context.Context, store *pkgskill.SkillStore) error {
	manifests, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("LoadFromCAS: failed to list skills: %w", err)
	}

	loaded := 0
	skipped := 0
	for _, m := range manifests {
		// Skip if already registered (directory-loaded skills take precedence)
		r.mu.RLock()
		_, exists := r.skills[m.Name]
		r.mu.RUnlock()
		if exists {
			continue
		}

		// Fetch content tar and extract SKILL.md
		contentTar, err := store.GetContent(ctx, m.Name, m.Version)
		if err != nil {
			slog.Warn("LoadFromCAS: skip skill, no content", "name", m.Name, "error", err)
			skipped++
			continue
		}

		skillMD, err := extractSkillMDFromTar(contentTar)
		if err != nil {
			slog.Warn("LoadFromCAS: skip skill, no SKILL.md in tar", "name", m.Name, "error", err)
			skipped++
			continue
		}

		// Parse the SKILL.md to get full frontmatter + body
		fmStr, body, err := extractFrontmatter(skillMD)
		if err != nil {
			// Use manifest data as fallback
			fmStr = ""
			body = skillMD
		}

		// Build SkillDefinition by combining manifest + parsed frontmatter
		def := &SkillDefinition{
			Name:        m.Name,
			Description: m.Description,
			Content:     body,
			Version:     m.Version,
			ParsedAt:    time.Now(),
			PluginName:  "cas",
		}

		// Parse frontmatter for rich fields (tier, agents, triggers, tool_deps)
		if fmStr != "" {
			var meta Metadata
			if yamlErr := yaml.Unmarshal([]byte(fmStr), &meta); yamlErr == nil {
				if len(meta.Triggers) > 0 {
					def.Triggers = meta.Triggers
				}
				if meta.ToolDependencies != nil {
					def.ToolDependencies = meta.ToolDependencies
				}
				if meta.Tier > 0 {
					def.Tier = meta.Tier
				}
				if len(meta.Agents) > 0 {
					def.Agents = meta.Agents
				}
				def.Portable = meta.Portable
				def.Hidden = meta.Hidden
			}
		}

		// Apply defaults for fields required by IsValid
		if len(def.Triggers) == 0 && m.Triggers != nil {
			def.Triggers = m.Triggers
		}
		if len(def.Triggers) == 0 {
			// Generate trigger from description first sentence
			parts := strings.SplitN(def.Description, ".", 2)
			if len(parts) > 0 && len(parts[0]) > 5 {
				def.Triggers = []string{strings.ToLower(strings.TrimSpace(parts[0]))}
			} else {
				def.Triggers = []string{def.Name}
			}
		}
		if def.Tier == 0 {
			def.Tier = 1
		}
		if len(def.Agents) == 0 {
			def.Agents = []string{"primary"}
		}

		if err := r.RegisterSkill(ctx, def); err != nil {
			slog.Debug("LoadFromCAS: skip skill", "name", m.Name, "error", err)
			skipped++
			continue
		}
		loaded++
	}

	slog.Info("LoadFromCAS complete", "loaded", loaded, "skipped", skipped, "total_manifests", len(manifests))
	return nil
}

// extractSkillMDFromTar reads a tar archive and returns the content of the first SKILL.md file found.
func extractSkillMDFromTar(tarBytes []byte) (string, error) {
	tr := tar.NewReader(bytes.NewReader(tarBytes))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar read error: %w", err)
		}
		if filepath.Base(hdr.Name) == "SKILL.md" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return "", fmt.Errorf("read SKILL.md from tar: %w", err)
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("SKILL.md not found in tar archive")
}
