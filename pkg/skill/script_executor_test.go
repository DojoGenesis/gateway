package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScriptExecutor(t *testing.T) {
	// With config
	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: "/tmp/scripts",
		Timeout: 10 * time.Second,
	})

	assert.NotNil(t, executor)
	assert.Equal(t, "/tmp/scripts", executor.baseDir)
	assert.Equal(t, 10*time.Second, executor.timeout)

	// Nil config (defaults)
	executor = NewScriptExecutor(nil)
	assert.NotNil(t, executor)
	assert.Equal(t, "/plugins", executor.baseDir)
	assert.Equal(t, 30*time.Second, executor.timeout)
}

func TestExecute_AllowedScript(t *testing.T) {
	// Create temp directory for test scripts
	tmpDir := t.TempDir()

	// Create a simple Python script
	scriptContent := `#!/usr/bin/env python3
import sys
print("Hello from Python!")
print("Args:", " ".join(sys.argv[1:]))
sys.exit(0)
`
	scriptPath := filepath.Join(tmpDir, "init_skill.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	result, err := executor.Execute(ctx, "init_skill.py", []string{"arg1", "arg2"})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result["status"])
	assert.Contains(t, result["stdout"], "Hello from Python")
	assert.Contains(t, result["stdout"], "arg1 arg2")
}

func TestExecute_DisallowedScript(t *testing.T) {
	tmpDir := t.TempDir()

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
	})

	ctx := context.Background()
	_, err := executor.Execute(ctx, "malicious_script.py", []string{})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScriptNotAllowed)
	assert.Contains(t, err.Error(), "malicious_script.py")
}

func TestExecute_ShellMetacharacters(t *testing.T) {
	tmpDir := t.TempDir()

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
	})

	ctx := context.Background()

	// Test dangerous characters
	dangerousArgs := []string{
		"arg; rm -rf /",
		"arg | cat /etc/passwd",
		"arg & malicious_command",
		"arg $(whoami)",
		"arg `whoami`",
		"arg\nmalicious",
		"arg > /tmp/exploit",
		"arg < /etc/shadow",
	}

	for _, arg := range dangerousArgs {
		t.Run(arg, func(t *testing.T) {
			_, err := executor.Execute(ctx, "init_skill.py", []string{arg})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "shell metacharacters")
		})
	}
}

func TestExecute_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
	})

	ctx := context.Background()

	// Attempt path traversal
	_, err := executor.Execute(ctx, "../../../etc/passwd", []string{})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScriptNotAllowed)
}

func TestExecute_Timeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that sleeps
	scriptContent := `#!/usr/bin/env python3
import time
time.sleep(10)
print("Done")
`
	scriptPath := filepath.Join(tmpDir, "init_skill.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
		Timeout: 100 * time.Millisecond,
	})

	ctx := context.Background()
	_, err = executor.Execute(ctx, "init_skill.py", []string{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
}

func TestExecute_ShellScript(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple shell script
	scriptContent := `#!/bin/bash
echo "Hello from Bash!"
echo "Args: $@"
exit 0
`
	scriptPath := filepath.Join(tmpDir, "smart_clone.sh")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	result, err := executor.Execute(ctx, "smart_clone.sh", []string{"arg1", "arg2"})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "success", result["status"])
	assert.Contains(t, result["stdout"], "Hello from Bash")
	assert.Contains(t, result["stdout"], "arg1 arg2")
}

func TestExecute_ScriptError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that exits with error
	scriptContent := `#!/usr/bin/env python3
import sys
print("Error message", file=sys.stderr)
sys.exit(1)
`
	scriptPath := filepath.Join(tmpDir, "init_skill.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
		Timeout: 5 * time.Second,
	})

	ctx := context.Background()
	result, err := executor.Execute(ctx, "init_skill.py", []string{})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrScriptExecutionFailed)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result["status"])
	assert.Contains(t, result["stdout"], "Error message")
}

func TestExecuteScript_ConvenienceMethod(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple script
	scriptContent := `#!/usr/bin/env python3
import sys
print("Args:", len(sys.argv) - 1)
`
	scriptPath := filepath.Join(tmpDir, "init_skill.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
	})

	ctx := context.Background()
	result, err := executor.ExecuteScript(ctx, "init_skill.py", "arg1", "arg2", "arg3")

	assert.NoError(t, err)
	assert.Contains(t, result["stdout"], "Args: 3")
}

func TestIsScriptAllowed(t *testing.T) {
	executor := NewScriptExecutor(nil)

	// Allowed scripts
	assert.True(t, executor.IsScriptAllowed("init_skill.py"))
	assert.True(t, executor.IsScriptAllowed("suggest_seeds.py"))
	assert.True(t, executor.IsScriptAllowed("smart_clone.sh"))
	assert.True(t, executor.IsScriptAllowed("lychee"))

	// Disallowed scripts
	assert.False(t, executor.IsScriptAllowed("malicious.py"))
	assert.False(t, executor.IsScriptAllowed("unknown.sh"))
}

func TestGetAllowedScripts(t *testing.T) {
	executor := NewScriptExecutor(nil)

	scripts := executor.GetAllowedScripts()

	assert.NotEmpty(t, scripts)
	assert.Contains(t, scripts, "init_skill.py")
	assert.Contains(t, scripts, "suggest_seeds.py")
	assert.Contains(t, scripts, "smart_clone.sh")
	assert.Contains(t, scripts, "lychee")
	assert.Len(t, scripts, 8) // Should match AllowedScripts count
}

func TestHasShellMetachars(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"normal_arg", false},
		{"arg-with-dash", false},
		{"arg_with_underscore", false},
		{"arg/with/slash", false},
		{"arg.with.dot", false},
		{"arg with space", false}, // Spaces are OK (will be quoted by exec)
		{"arg;malicious", true},
		{"arg|pipe", true},
		{"arg&background", true},
		{"arg$(command)", true},
		{"arg`command`", true},
		{"arg\nmalicious", true},
		{"arg>redirect", true},
		{"arg<redirect", true},
		{"arg(subshell)", true},
		{"arg{brace}", true},
		{"arg*glob", true},
		{"arg?glob", true},
		{"arg[glob]", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := hasShellMetachars(tt.input)
			assert.Equal(t, tt.expected, result, "Input: %s", tt.input)
		})
	}
}

func TestExecute_EmptyArgs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a script that works with no args
	scriptContent := `#!/usr/bin/env python3
print("No args needed")
`
	scriptPath := filepath.Join(tmpDir, "init_skill.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
	})

	ctx := context.Background()
	result, err := executor.Execute(ctx, "init_skill.py", []string{})

	assert.NoError(t, err)
	assert.Equal(t, "success", result["status"])
	assert.Contains(t, result["stdout"], "No args needed")
}

func TestExecute_NilArgs(t *testing.T) {
	tmpDir := t.TempDir()

	scriptContent := `#!/usr/bin/env python3
print("Test")
`
	scriptPath := filepath.Join(tmpDir, "init_skill.py")
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	executor := NewScriptExecutor(&ScriptExecutorConfig{
		BaseDir: tmpDir,
	})

	ctx := context.Background()
	result, err := executor.Execute(ctx, "init_skill.py", nil)

	assert.NoError(t, err)
	assert.Equal(t, "success", result["status"])
}
