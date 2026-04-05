// Package skill implements the Skill Marketplace — OCI-compatible distribution
// for agent skills stored in content-addressable storage (CAS).
//
// Skills are packaged as two CAS blobs: a config blob (JSON manifest) and a
// content blob (tar archive of skill files). Tags follow the pattern
// "skill/{name}" with the version string for OCI-compatible resolution.
package skill

// SkillManifest describes a packaged skill's metadata.
// It is serialized to JSON as the config blob in the CAS-backed OCI layer.
type SkillManifest struct {
	// Name is the unique skill identifier (e.g. "strategic-scout").
	Name string `json:"name"`

	// Version follows semver (e.g. "1.0.0").
	Version string `json:"version"`

	// Description is a one-line summary of what the skill does.
	Description string `json:"description"`

	// Triggers lists phrases or patterns that activate the skill.
	Triggers []string `json:"triggers,omitempty"`

	// Dependencies lists other skills this skill depends on.
	Dependencies []string `json:"dependencies,omitempty"`

	// License is the SPDX license identifier (e.g. "MIT").
	License string `json:"license,omitempty"`

	// Authors lists the skill's creators.
	Authors []string `json:"authors,omitempty"`

	// Platform holds platform-specific constraints (e.g. "os": "linux").
	Platform map[string]string `json:"platform,omitempty"`

	// Inputs declares typed input ports for workflow connections (Era 3).
	Inputs []PortDefinition `json:"inputs,omitempty"`

	// Outputs declares typed output ports for workflow connections (Era 3).
	Outputs []PortDefinition `json:"outputs,omitempty"`
}

// PortDefinition describes a typed input or output port on a skill.
// Used by the workflow builder to validate connections at drag time.
type PortDefinition struct {
	// Name identifies the port (e.g. "sources", "format").
	Name string `json:"name"`

	// Type is the port's data type: "string", "number", "boolean", "string[]", "object", "any".
	Type string `json:"type"`

	// Description explains what this port carries.
	Description string `json:"description,omitempty"`

	// Required marks whether the input must be connected.
	Required bool `json:"required,omitempty"`

	// Default is the fallback value when the port is unconnected.
	Default any `json:"default,omitempty"`

	// Enum constrains string values to a fixed set.
	Enum []string `json:"enum,omitempty"`
}

// PluginManifest extends SkillManifest for Era 3 distributable packages.
// A plugin is the OCI distribution unit that may contain skills, workflows,
// WASM modules, or channel adapters.
type PluginManifest struct {
	SkillManifest

	// PluginType categorizes the package: "skill", "workflow", "adapter", "bundle".
	PluginType string `json:"plugin_type,omitempty"`

	// TrustTier indicates the signing level: 0=community, 1=verified, 2=official.
	TrustTier int `json:"trust_tier,omitempty"`

	// Contents lists the artifacts contained in this plugin.
	Contents []ContentEntry `json:"contents,omitempty"`

	// Yanked marks this version as withdrawn from new installs.
	Yanked bool `json:"yanked,omitempty"`

	// YankReason explains why the version was yanked.
	YankReason string `json:"yank_reason,omitempty"`
}

// ContentEntry describes one artifact inside a plugin package.
type ContentEntry struct {
	// Type is the artifact kind: "skill", "workflow", "wasm-module", "channel-adapter".
	Type string `json:"type"`

	// Name identifies the artifact.
	Name string `json:"name"`

	// Path is the relative path within the tar archive.
	Path string `json:"path"`
}

// PluginArtifactType is the OCI artifact type for plugin packages.
const PluginArtifactType = "application/vnd.dojo.plugin.v1"

// OCI media types for skill artifacts.
const (
	// ArtifactType is the OCI artifact type for skill packages.
	ArtifactType = "application/vnd.dojo.skill.v1"

	// ConfigMediaType is the media type for the skill manifest JSON.
	ConfigMediaType = "application/vnd.dojo.skill.config.v1+json"

	// ContentMediaType is the media type for the skill content tar archive.
	ContentMediaType = "application/vnd.dojo.skill.content.v1+tar"
)
