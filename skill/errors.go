package skill

import "errors"

var (
	// Registry errors
	ErrSkillNotFound           = errors.New("skill not found")
	ErrSkillAlreadyExists      = errors.New("skill already registered")
	ErrInvalidSkillName        = errors.New("skill name cannot be empty")
	ErrInvalidSkillDescription = errors.New("skill description cannot be empty")
	ErrInvalidSkillTriggers    = errors.New("skill must have at least one trigger phrase")
	ErrInvalidSkillTier        = errors.New("skill tier must be 1-4")
	ErrInvalidSkillAgents      = errors.New("non-hidden skills must specify at least one agent")
	ErrInvalidToolDependency   = errors.New("invalid tool dependency")

	// Parsing errors
	ErrInvalidYAMLFrontmatter = errors.New("invalid YAML frontmatter")
	ErrMissingFrontmatter     = errors.New("SKILL.md missing YAML frontmatter")
	ErrInvalidSkillFile       = errors.New("invalid SKILL.md file")

	// Execution errors
	ErrSkillExecutionFailed  = errors.New("skill execution failed")
	ErrAdapterNotAvailable   = errors.New("required adapter not available")
	ErrScriptNotAllowed      = errors.New("script not in allowlist")
	ErrScriptExecutionFailed = errors.New("script execution failed")
	ErrMaxDepthExceeded      = errors.New("max call depth exceeded")

	// Version errors
	ErrVersionMismatch = errors.New("gateway version does not meet skill requirements")
)
