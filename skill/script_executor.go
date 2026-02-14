package skill

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AllowedScripts is the security allowlist of scripts that can be executed
var AllowedScripts = map[string]bool{
	"init_skill.py":     true,
	"suggest_seeds.py":  true,
	"diff_tracker.py":   true,
	"context_mapper.py": true,
	"smart_clone.sh":    true,
	"apply_seed.py":     true,
	"lychee":            true,
	"validate_skill.py": true, // Additional script for skill validation
}

// ScriptExecutor safely executes allowed scripts from skills
type ScriptExecutor struct {
	baseDir string        // Base directory for script execution (e.g., /plugins/)
	timeout time.Duration // Maximum execution time per script
}

// ScriptExecutorConfig configures the script executor
type ScriptExecutorConfig struct {
	BaseDir string
	Timeout time.Duration
}

// NewScriptExecutor creates a new script executor
func NewScriptExecutor(config *ScriptExecutorConfig) *ScriptExecutor {
	if config == nil {
		config = &ScriptExecutorConfig{
			BaseDir: "/plugins",
			Timeout: 30 * time.Second,
		}
	}

	// Default timeout if not specified
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Default base dir if not specified
	baseDir := config.BaseDir
	if baseDir == "" {
		baseDir = "/plugins"
	}

	return &ScriptExecutor{
		baseDir: baseDir,
		timeout: timeout,
	}
}

// Execute runs an allowed script with given arguments
func (s *ScriptExecutor) Execute(ctx context.Context, scriptName string, args []string) (map[string]interface{}, error) {
	// 1. Verify script is in allowlist
	if !AllowedScripts[scriptName] {
		return nil, fmt.Errorf("%w: %s (allowed: %v)", ErrScriptNotAllowed, scriptName, getAllowedScriptNames())
	}

	// 2. Find script in baseDir
	scriptPath := filepath.Join(s.baseDir, scriptName)

	// 3. Validate path doesn't escape base directory
	absScriptPath, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script path: %w", err)
	}

	absBaseDir, err := filepath.Abs(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base directory: %w", err)
	}

	if !strings.HasPrefix(absScriptPath, absBaseDir) {
		return nil, fmt.Errorf("script path escapes base directory: %s", scriptPath)
	}

	// 4. Validate arguments (no shell metacharacters)
	for i, arg := range args {
		if hasShellMetachars(arg) {
			return nil, fmt.Errorf("argument %d contains shell metacharacters: %s", i, arg)
		}
	}

	// 5. Execute with timeout
	execCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Determine interpreter based on file extension
	var cmd *exec.Cmd
	if strings.HasSuffix(scriptName, ".py") {
		// Python script
		cmdArgs := append([]string{absScriptPath}, args...)
		cmd = exec.CommandContext(execCtx, "python3", cmdArgs...)
	} else if strings.HasSuffix(scriptName, ".sh") {
		// Shell script
		cmdArgs := append([]string{absScriptPath}, args...)
		cmd = exec.CommandContext(execCtx, "bash", cmdArgs...)
	} else {
		// Assume it's an executable binary (e.g., lychee)
		cmd = exec.CommandContext(execCtx, absScriptPath, args...)
	}

	// Capture combined output
	output, execErr := cmd.CombinedOutput()

	// Check for timeout
	if execCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("script execution timeout (%s): %s", s.timeout, scriptName)
	}

	// Build result
	result := map[string]interface{}{
		"script": scriptName,
		"stdout": string(output),
		"args":   args,
	}

	if execErr != nil {
		result["status"] = "error"
		result["error"] = execErr.Error()
		return result, fmt.Errorf("%w: %s: %v", ErrScriptExecutionFailed, scriptName, execErr)
	}

	result["status"] = "success"
	return result, nil
}

// ExecuteScript is a convenience method for executing scripts with string arguments
func (s *ScriptExecutor) ExecuteScript(ctx context.Context, scriptName string, args ...string) (map[string]interface{}, error) {
	return s.Execute(ctx, scriptName, args)
}

// IsScriptAllowed returns true if the script is in the allowlist
func (s *ScriptExecutor) IsScriptAllowed(scriptName string) bool {
	return AllowedScripts[scriptName]
}

// GetAllowedScripts returns a list of all allowed script names
func (s *ScriptExecutor) GetAllowedScripts() []string {
	return getAllowedScriptNames()
}

// hasShellMetachars checks if a string contains shell metacharacters
func hasShellMetachars(s string) bool {
	// Dangerous shell metacharacters that could enable command injection
	dangerousChars := []string{
		"|",  // Pipe
		";",  // Command separator
		"&",  // Background execution
		"$",  // Variable expansion
		"`",  // Command substitution
		"\n", // Newline
		"\r", // Carriage return
		"<",  // Input redirection
		">",  // Output redirection
		"(",  // Subshell
		")",  // Subshell
		"{",  // Brace expansion
		"}",  // Brace expansion
		"*",  // Glob
		"?",  // Glob
		"[",  // Glob
		"]",  // Glob
	}

	for _, char := range dangerousChars {
		if strings.Contains(s, char) {
			return true
		}
	}

	return false
}

// getAllowedScriptNames returns a sorted list of allowed script names
func getAllowedScriptNames() []string {
	names := make([]string, 0, len(AllowedScripts))
	for name := range AllowedScripts {
		names = append(names, name)
	}
	return names
}
