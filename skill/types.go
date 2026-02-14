package skill

import (
	"time"
)

// SkillDefinition represents a registered skill with metadata.
// Skills are portable, reusable patterns that can be invoked by agents
// or other skills to perform complex workflows.
type SkillDefinition struct {
	// ID is a unique UUID assigned at registration
	ID string `json:"id"`

	// Name is the skill's slug identifier (kebab-case)
	Name string `json:"name"`

	// Description is a 1-sentence summary of the skill's purpose
	Description string `json:"description"`

	// PluginName identifies which plugin this skill belongs to
	PluginName string `json:"plugin_name"`

	// FilePath is the absolute path to the SKILL.md file
	FilePath string `json:"file_path"`

	// Triggers is a list of invocation phrases that trigger this skill
	Triggers []string `json:"triggers"`

	// ToolDependencies lists required tool types: file_system, bash, web_tools, script_execution, meta_skill
	ToolDependencies []string `json:"tool_dependencies"`

	// Tier indicates portability level (1-4):
	// 1 = Zero-change port (file_system, bash only)
	// 2 = Requires adapters (web_tools, script_execution)
	// 3 = Meta-skills (skill invocation, DAG composition)
	// 4 = Complex integration (requires refactoring)
	Tier int `json:"tier"`

	// Portable indicates if skill is tool-agnostic (true) or Claude-locked (false)
	Portable bool `json:"portable"`

	// Agents lists which agents can invoke this skill
	Agents []string `json:"agents"`

	// RequiredVersion specifies minimum gateway version (e.g., "0.3.0")
	RequiredVersion string `json:"required_version,omitempty"`

	// PythonScripts lists Python scripts required by this skill
	PythonScripts []string `json:"python_scripts,omitempty"`

	// ShellScripts lists shell scripts required by this skill
	ShellScripts []string `json:"shell_scripts,omitempty"`

	// Hidden indicates if skill is only invokable by other skills (not agents)
	Hidden bool `json:"hidden"`

	// Content is the raw markdown body (without YAML frontmatter)
	Content string `json:"content"`

	// ParsedAt is when the YAML frontmatter was parsed
	ParsedAt time.Time `json:"parsed_at"`

	// LoadedAt is when the skill was registered
	LoadedAt time.Time `json:"loaded_at"`

	// Version from YAML metadata
	Version string `json:"version,omitempty"`

	// Created timestamp from YAML metadata
	Created string `json:"created,omitempty"`

	// Author from YAML metadata
	Author string `json:"author,omitempty"`
}

// MetadataBlock represents the nested metadata section (per spec)
type MetadataBlock struct {
	Version          string   `yaml:"version,omitempty"`
	Created          string   `yaml:"created,omitempty"`
	Author           string   `yaml:"author,omitempty"`
	ToolDependencies []string `yaml:"tool_dependencies"`
	Portable         bool     `yaml:"portable"`
	Tier             int      `yaml:"tier"`
	Agents           []string `yaml:"agents"`
	RequiresVersion  string   `yaml:"requires_version,omitempty"`
	PythonScripts    []string `yaml:"python_scripts,omitempty"`
	ShellScripts     []string `yaml:"shell_scripts,omitempty"`
	Hidden           bool     `yaml:"hidden,omitempty"`
}

// Metadata represents the YAML frontmatter structure in SKILL.md files
// Supports both nested (spec-compliant) and flat (backward compat) formats
type Metadata struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`

	// Nested format (per spec) - takes precedence if present
	MetadataBlock *MetadataBlock `yaml:"metadata,omitempty"`

	// Flat format fields (backward compatibility - only used if metadata block is nil)
	ToolDependencies []string               `yaml:"tool_dependencies,omitempty"`
	Tier             int                    `yaml:"tier,omitempty"`
	Portable         bool                   `yaml:"portable,omitempty"`
	Agents           []string               `yaml:"agents,omitempty"`
	RequiredVersion  string                 `yaml:"required_version,omitempty"`
	PythonScripts    []string               `yaml:"python_scripts,omitempty"`
	ShellScripts     []string               `yaml:"shell_scripts,omitempty"`
	Hidden           bool                   `yaml:"hidden,omitempty"`
	Version          string                 `yaml:"version,omitempty"`
	Created          string                 `yaml:"created,omitempty"`
	Author           string                 `yaml:"author,omitempty"`
	Extra            map[string]interface{} `yaml:",inline"` // Catch-all for unknown fields
}

// ValidToolDependencies is the allowlist of valid tool dependency types
var ValidToolDependencies = map[string]bool{
	"file_system":      true,
	"bash":             true,
	"web_tools":        true,
	"script_execution": true,
	"meta_skill":       true,
}

// IsValid validates that the SkillDefinition has all required fields
func (s *SkillDefinition) IsValid() error {
	if s.Name == "" {
		return ErrInvalidSkillName
	}
	if s.Description == "" {
		return ErrInvalidSkillDescription
	}
	if len(s.Triggers) == 0 {
		return ErrInvalidSkillTriggers
	}
	if s.Tier < 1 || s.Tier > 4 {
		return ErrInvalidSkillTier
	}
	if !s.Hidden && len(s.Agents) == 0 {
		return ErrInvalidSkillAgents
	}
	for _, dep := range s.ToolDependencies {
		if !ValidToolDependencies[dep] {
			return ErrInvalidToolDependency
		}
	}
	return nil
}

// RequiresAdapter returns true if the skill needs an adapter (Tier 2+)
func (s *SkillDefinition) RequiresAdapter() bool {
	return s.Tier >= 2
}

// RequiresWebTools returns true if the skill depends on web_tools
func (s *SkillDefinition) RequiresWebTools() bool {
	for _, dep := range s.ToolDependencies {
		if dep == "web_tools" {
			return true
		}
	}
	return false
}

// RequiresScriptExecution returns true if the skill depends on script_execution
func (s *SkillDefinition) RequiresScriptExecution() bool {
	for _, dep := range s.ToolDependencies {
		if dep == "script_execution" {
			return true
		}
	}
	return false
}

// IsMetaSkill returns true if the skill can invoke other skills (Tier 3+)
func (s *SkillDefinition) IsMetaSkill() bool {
	return s.Tier >= 3
}
