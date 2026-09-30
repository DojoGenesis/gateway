package server

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DojoGenesis/gateway/server/middleware"
)

// The workflow run_command refusal read only the ENVIRONMENT env var. On a
// host whose production setting lives in config.yaml alone, a workflow step
// could shell out by default.
func TestProductionCheck_YAMLOnlyProductionBlocksRunCommand(t *testing.T) {
	t.Setenv("ENVIRONMENT", "")
	t.Setenv(EnvWorkflowRunCommand, "")
	middleware.SetResolvedEnvironment("production")
	t.Cleanup(func() { middleware.SetResolvedEnvironment("") })

	marker, err := runTouch(t)

	require.Error(t, err, "a YAML-configured production gateway must refuse run_command by default")
	assert.False(t, fileExists(t, marker), "the command still executed")
}

// gin's mode used an exact == "production".
func TestProductionCheck_GinReleaseModeUsesTheSharedPredicate(t *testing.T) {
	for _, env := range []string{"production", "Production", " PRODUCTION "} {
		assert.Equal(t, gin.ReleaseMode, ginModeFor(env), "environment %q", env)
	}
	assert.Equal(t, gin.DebugMode, ginModeFor("development"))
}
