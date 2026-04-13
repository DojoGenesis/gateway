package server

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/DojoGenesis/gateway/tools"
	"github.com/DojoGenesis/gateway/workflow"
)

// Ensure interface compliance at compile time.
var _ workflow.SkillRunner = (*CommandSkillRunner)(nil)

// CommandSkillRunner bridges workflow steps to the run_command tool.
// When a workflow step specifies skill="run_command", it dispatches
// to tools.RunCommand with the step's inputs as parameters.
type CommandSkillRunner struct{}

// NewCommandSkillRunner returns a SkillRunner that dispatches to tools.RunCommand.
func NewCommandSkillRunner() *CommandSkillRunner {
	return &CommandSkillRunner{}
}

// RunSkill executes a named skill. Currently only "run_command" is supported;
// all other skill names return an error.
func (r *CommandSkillRunner) RunSkill(ctx context.Context, skillName string, input map[string]string) (string, error) {
	switch skillName {
	case "run_command":
		return r.runCommand(ctx, input)
	default:
		return "", fmt.Errorf("skill runner: unsupported skill %q (only run_command is implemented)", skillName)
	}
}

// runCommand converts string inputs to the map[string]interface{} format
// expected by tools.RunCommand and returns stdout on success.
func (r *CommandSkillRunner) runCommand(ctx context.Context, input map[string]string) (string, error) {
	command := input["command"]
	if command == "" {
		return "", fmt.Errorf("run_command: command input is required")
	}

	params := map[string]interface{}{
		"command": command,
	}

	if wd, ok := input["working_directory"]; ok && wd != "" {
		params["working_directory"] = wd
	}

	if timeoutStr, ok := input["timeout"]; ok && timeoutStr != "" {
		// tools.RunCommand caps timeout at 300s (5 minutes). Parse the value
		// and clamp before passing so RunCommand doesn't reject it outright.
		if secs, err := strconv.ParseFloat(timeoutStr, 64); err == nil {
			if secs > 300 {
				secs = 300
			}
			params["timeout"] = secs
		}
	}

	slog.Info("workflow: executing run_command",
		"command", command,
		"working_directory", input["working_directory"],
	)

	start := time.Now()
	result, err := tools.RunCommand(ctx, params)
	elapsed := time.Since(start)

	if err != nil {
		return "", fmt.Errorf("run_command failed: %w", err)
	}

	// tools.RunCommand returns {"success": bool, "stdout": string, "stderr": string, "exit_code": int}
	success, _ := result["success"].(bool)
	stdout, _ := result["stdout"].(string)
	stderr, _ := result["stderr"].(string)
	exitCode, _ := result["exit_code"].(int)

	slog.Info("workflow: run_command completed",
		"command", command,
		"success", success,
		"exit_code", exitCode,
		"elapsed", elapsed,
	)

	if !success {
		return stdout, fmt.Errorf("run_command exited %d: %s", exitCode, stderr)
	}

	return stdout, nil
}
