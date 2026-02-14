package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func RunCommand(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "command parameter is required",
		}, nil
	}

	workingDir := GetStringParam(params, "working_directory", ".")
	timeout := GetDurationParam(params, "timeout", 30*time.Second)
	shell := GetStringParam(params, "shell", getDefaultShell())

	if timeout > 5*time.Minute {
		return map[string]interface{}{
			"success": false,
			"error":   "timeout cannot exceed 5 minutes for security reasons",
		}, nil
	}

	if err := validateCommand(command); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("command validation failed: %v", err),
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, shell, "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}
	cmd.Dir = workingDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	err := cmd.Run()
	executionTime := time.Since(startTime)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			return map[string]interface{}{
				"success":        false,
				"command":        command,
				"stdout":         stdout.String(),
				"stderr":         stderr.String(),
				"error":          fmt.Sprintf("command timed out after %v", timeout),
				"exit_code":      -1,
				"execution_time": executionTime.Seconds(),
				"timed_out":      true,
			}, nil
		} else {
			return map[string]interface{}{
				"success":        false,
				"command":        command,
				"stdout":         stdout.String(),
				"stderr":         stderr.String(),
				"error":          err.Error(),
				"exit_code":      -1,
				"execution_time": executionTime.Seconds(),
			}, nil
		}
	}

	return map[string]interface{}{
		"success":           exitCode == 0,
		"command":           command,
		"stdout":            stdout.String(),
		"stderr":            stderr.String(),
		"exit_code":         exitCode,
		"execution_time":    executionTime.Seconds(),
		"working_directory": workingDir,
	}, nil
}

func getDefaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

func validateCommand(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	commandLower := strings.ToLower(command)

	if (strings.Contains(commandLower, "rm -rf /") && !strings.Contains(commandLower, "rm -rf //")) ||
		strings.Contains(commandLower, "rm -rf /*") {
		if !strings.Contains(commandLower, "rm -rf /tmp") &&
			!strings.Contains(commandLower, "rm -rf /var") &&
			!strings.Contains(commandLower, "rm -rf /home") &&
			!strings.Contains(commandLower, "rm -rf /users") {
			return fmt.Errorf("potentially dangerous command pattern detected: rm -rf on system root")
		}
	}

	dangerousPatterns := []string{
		"mkfs",
		"dd if=/dev/zero",
		":(){ :|:& };:",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(commandLower, strings.ToLower(pattern)) {
			return fmt.Errorf("potentially dangerous command pattern detected: %s", pattern)
		}
	}

	if strings.Contains(commandLower, "chmod") &&
		(strings.Contains(commandLower, "777") || strings.Contains(commandLower, "666")) &&
		(strings.Contains(commandLower, " / ") || strings.Contains(commandLower, "\t/ ") ||
			strings.HasSuffix(commandLower, " /") || strings.HasSuffix(commandLower, "\t/")) {
		return fmt.Errorf("potentially dangerous command pattern detected: chmod 777/666 on root")
	}

	return nil
}

func init() {
	RegisterTool(&ToolDefinition{
		Name:        "run_command",
		Description: "Execute a shell command and capture its output. WARNING: Use with caution as this can execute arbitrary commands on the system.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The shell command to execute",
				},
				"working_directory": map[string]interface{}{
					"type":        "string",
					"description": "The working directory to execute the command in (default: current directory)",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Timeout in seconds (default: 30, max: 300)",
				},
				"shell": map[string]interface{}{
					"type":        "string",
					"description": "Shell to use for execution (default: sh on Unix, cmd on Windows)",
				},
			},
			"required": []string{"command"},
		},
		Function: RunCommand,
	})
}
