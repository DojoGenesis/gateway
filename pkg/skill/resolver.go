package skill

import (
	"fmt"
	"strings"
)

// DefaultRegistry is the default OCI registry for skill:// references.
const DefaultRegistry = "ghcr.io"

// DefaultNamespace is the default namespace under the registry.
const DefaultNamespace = "dojo-skills"

// ResolvedRef contains the parsed reference information for a skill.
type ResolvedRef struct {
	// Registry is the OCI registry host (e.g. "ghcr.io").
	Registry string

	// Path is the repository path within the registry (e.g. "dojo-skills/strategic-scout").
	Path string

	// Tag is the version or tag string (e.g. "1.0.0", "latest").
	Tag string

	// Scheme identifies the reference type: "skill", "oci", or "github".
	Scheme string
}

// Resolve parses a skill reference URL into its components.
//
// Supported schemes:
//
//	skill://name@version  ->  maps to default registry (ghcr.io/dojo-skills/name:version)
//	oci://registry/path:tag  ->  direct OCI reference
//	github:org/repo//path  ->  GitHub-hosted skill
//
// If no scheme is provided, it is treated as a skill:// reference.
func Resolve(ref string) (ResolvedRef, error) {
	if ref == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty reference")
	}

	switch {
	case strings.HasPrefix(ref, "skill://"):
		return resolveSkillScheme(ref)
	case strings.HasPrefix(ref, "oci://"):
		return resolveOCIScheme(ref)
	case strings.HasPrefix(ref, "github:"):
		return resolveGitHubScheme(ref)
	default:
		// Treat bare references as skill:// scheme.
		return resolveSkillScheme("skill://" + ref)
	}
}

// resolveSkillScheme parses skill://name@version references.
func resolveSkillScheme(ref string) (ResolvedRef, error) {
	body := strings.TrimPrefix(ref, "skill://")
	if body == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty skill name in %q", ref)
	}

	name := body
	version := "latest"

	if idx := strings.LastIndex(body, "@"); idx > 0 {
		name = body[:idx]
		version = body[idx+1:]
		if version == "" {
			version = "latest"
		}
	}

	if name == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty skill name in %q", ref)
	}

	return ResolvedRef{
		Registry: DefaultRegistry,
		Path:     DefaultNamespace + "/" + name,
		Tag:      version,
		Scheme:   "skill",
	}, nil
}

// resolveOCIScheme parses oci://registry/path:tag references.
func resolveOCIScheme(ref string) (ResolvedRef, error) {
	body := strings.TrimPrefix(ref, "oci://")
	if body == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty OCI reference in %q", ref)
	}

	// Split registry from path at the first slash.
	slashIdx := strings.Index(body, "/")
	if slashIdx < 0 {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: OCI reference %q missing path (expected registry/path:tag)", ref)
	}

	registry := body[:slashIdx]
	pathAndTag := body[slashIdx+1:]

	if registry == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty registry in %q", ref)
	}

	// Split path:tag at the last colon.
	path := pathAndTag
	tag := "latest"

	if colonIdx := strings.LastIndex(pathAndTag, ":"); colonIdx > 0 {
		path = pathAndTag[:colonIdx]
		tag = pathAndTag[colonIdx+1:]
		if tag == "" {
			tag = "latest"
		}
	}

	if path == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty path in %q", ref)
	}

	return ResolvedRef{
		Registry: registry,
		Path:     path,
		Tag:      tag,
		Scheme:   "oci",
	}, nil
}

// resolveGitHubScheme parses github:org/repo//path references.
// The double-slash separates the repository from the skill path within it.
func resolveGitHubScheme(ref string) (ResolvedRef, error) {
	body := strings.TrimPrefix(ref, "github:")
	if body == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty GitHub reference in %q", ref)
	}

	// Split at // to separate repo from path.
	parts := strings.SplitN(body, "//", 2)
	repo := parts[0]
	skillPath := ""
	if len(parts) == 2 {
		skillPath = parts[1]
	}

	if repo == "" {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: empty repository in %q", ref)
	}

	// Validate org/repo format.
	if !strings.Contains(repo, "/") {
		return ResolvedRef{}, fmt.Errorf("skill: resolve: GitHub reference %q must be org/repo format", ref)
	}

	// Build the path: repo + skill path.
	fullPath := repo
	if skillPath != "" {
		fullPath = repo + "/" + skillPath
	}

	// Tag defaults to "main" for GitHub references.
	tag := "main"

	// Check for @version suffix on the skill path.
	if idx := strings.LastIndex(fullPath, "@"); idx > 0 {
		tag = fullPath[idx+1:]
		fullPath = fullPath[:idx]
		if tag == "" {
			tag = "main"
		}
	}

	return ResolvedRef{
		Registry: "github.com",
		Path:     fullPath,
		Tag:      tag,
		Scheme:   "github",
	}, nil
}
