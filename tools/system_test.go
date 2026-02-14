package tools

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunCommand_Success(t *testing.T) {
	ctx := context.Background()

	var command string
	if runtime.GOOS == "windows" {
		command = "echo hello"
	} else {
		command = "echo hello"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if !result["success"].(bool) {
		t.Errorf("Expected success=true, got false")
	}

	stdout := result["stdout"].(string)
	if !strings.Contains(stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got: %s", stdout)
	}

	exitCode := result["exit_code"].(int)
	if exitCode != 0 {
		t.Errorf("Expected exit_code=0, got %d", exitCode)
	}
}

func TestRunCommand_StdoutStderr(t *testing.T) {
	ctx := context.Background()

	var command string
	if runtime.GOOS == "windows" {
		command = "echo stdout && echo stderr 1>&2"
	} else {
		command = "echo stdout && echo stderr >&2"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	stdout := result["stdout"].(string)
	stderr := result["stderr"].(string)

	if !strings.Contains(stdout, "stdout") {
		t.Errorf("Expected stdout to contain 'stdout', got: %s", stdout)
	}

	if !strings.Contains(stderr, "stderr") {
		t.Errorf("Expected stderr to contain 'stderr', got: %s", stderr)
	}
}

func TestRunCommand_NonZeroExitCode(t *testing.T) {
	ctx := context.Background()

	var command string
	if runtime.GOOS == "windows" {
		command = "exit 42"
	} else {
		command = "exit 42"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if result["success"].(bool) {
		t.Errorf("Expected success=false for non-zero exit code")
	}

	exitCode := result["exit_code"].(int)
	if exitCode != 42 {
		t.Errorf("Expected exit_code=42, got %d", exitCode)
	}
}

func TestRunCommand_Timeout(t *testing.T) {
	ctx := context.Background()

	var command string
	if runtime.GOOS == "windows" {
		command = "timeout /t 5"
	} else {
		command = "sleep 5"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
		"timeout": 1,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if result["success"].(bool) {
		t.Errorf("Expected success=false for timeout")
	}

	if timedOut, ok := result["timed_out"].(bool); ok && !timedOut {
		t.Errorf("Expected timed_out=true")
	}

	if errorMsg, ok := result["error"].(string); ok {
		if !strings.Contains(errorMsg, "timed out") {
			t.Errorf("Expected error message to contain 'timed out', got: %s", errorMsg)
		}
	}
}

func TestRunCommand_EmptyCommand(t *testing.T) {
	ctx := context.Background()

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": "",
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if result["success"].(bool) {
		t.Errorf("Expected success=false for empty command")
	}

	errorMsg := result["error"].(string)
	if !strings.Contains(errorMsg, "required") {
		t.Errorf("Expected error about required parameter, got: %s", errorMsg)
	}
}

func TestRunCommand_MissingCommand(t *testing.T) {
	ctx := context.Background()

	result, err := RunCommand(ctx, map[string]interface{}{})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if result["success"].(bool) {
		t.Errorf("Expected success=false for missing command")
	}
}

func TestRunCommand_DangerousCommands(t *testing.T) {
	ctx := context.Background()

	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs.ext4 /dev/sda",
		"dd if=/dev/zero of=/dev/sda",
		":(){ :|:& };:",
		"chmod -R 777 /",
	}

	for _, cmd := range dangerousCommands {
		t.Run(cmd, func(t *testing.T) {
			result, err := RunCommand(ctx, map[string]interface{}{
				"command": cmd,
				"timeout": 1,
			})

			if err != nil {
				t.Fatalf("RunCommand failed: %v", err)
			}

			if result["success"].(bool) {
				t.Errorf("Expected success=false for dangerous command: %s", cmd)
			}

			if errorMsg, ok := result["error"].(string); ok {
				if !strings.Contains(errorMsg, "dangerous") && !strings.Contains(errorMsg, "timed out") {
					t.Errorf("Expected error about dangerous command or timeout, got: %s", errorMsg)
				}
			}
		})
	}
}

func TestRunCommand_WorkingDirectory(t *testing.T) {
	ctx := context.Background()

	var command string
	if runtime.GOOS == "windows" {
		command = "cd"
	} else {
		command = "pwd"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command":           command,
		"working_directory": "/tmp",
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if !result["success"].(bool) {
		t.Errorf("Expected success=true, got false. Error: %v", result["error"])
	}

	stdout := result["stdout"].(string)
	if runtime.GOOS != "windows" && !strings.Contains(stdout, "/tmp") {
		t.Errorf("Expected stdout to contain '/tmp', got: %s", stdout)
	}
}

func TestRunCommand_ExecutionTime(t *testing.T) {
	ctx := context.Background()

	var command string
	if runtime.GOOS == "windows" {
		command = "echo test"
	} else {
		command = "echo test"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	executionTime, ok := result["execution_time"].(float64)
	if !ok {
		t.Errorf("Expected execution_time to be float64")
	}

	if executionTime < 0 {
		t.Errorf("Expected execution_time >= 0, got %f", executionTime)
	}
}

func TestRunCommand_MaxTimeout(t *testing.T) {
	ctx := context.Background()

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": "echo test",
		"timeout": 301,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if result["success"].(bool) {
		t.Errorf("Expected success=false for timeout > 5 minutes")
	}

	errorMsg := result["error"].(string)
	if !strings.Contains(errorMsg, "cannot exceed 5 minutes") {
		t.Errorf("Expected error about max timeout, got: %s", errorMsg)
	}
}

func TestRunCommand_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	var command string
	if runtime.GOOS == "windows" {
		command = "timeout /t 10"
	} else {
		command = "sleep 10"
	}

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if result["success"].(bool) {
		t.Errorf("Expected success=false for cancelled context")
	}
}

func TestRunCommand_CustomShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping custom shell test on Windows")
	}

	ctx := context.Background()

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": "echo $0",
		"shell":   "bash",
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if !result["success"].(bool) {
		t.Errorf("Expected success=true, got false. Error: %v", result["error"])
	}
}

func TestRunCommand_MultilineCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping multiline test on Windows")
	}

	ctx := context.Background()

	command := `
		echo "line 1"
		echo "line 2"
		echo "line 3"
	`

	result, err := RunCommand(ctx, map[string]interface{}{
		"command": command,
	})

	if err != nil {
		t.Fatalf("RunCommand failed: %v", err)
	}

	if !result["success"].(bool) {
		t.Errorf("Expected success=true, got false. Error: %v", result["error"])
	}

	stdout := result["stdout"].(string)
	if !strings.Contains(stdout, "line 1") ||
		!strings.Contains(stdout, "line 2") ||
		!strings.Contains(stdout, "line 3") {
		t.Errorf("Expected stdout to contain all lines, got: %s", stdout)
	}
}

func TestGetDefaultShell(t *testing.T) {
	shell := getDefaultShell()

	if runtime.GOOS == "windows" {
		if shell != "cmd" {
			t.Errorf("Expected default shell to be 'cmd' on Windows, got: %s", shell)
		}
	} else {
		if shell != "sh" {
			t.Errorf("Expected default shell to be 'sh' on Unix, got: %s", shell)
		}
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		wantError bool
	}{
		{
			name:      "valid command",
			command:   "echo hello",
			wantError: false,
		},
		{
			name:      "empty command",
			command:   "",
			wantError: true,
		},
		{
			name:      "whitespace only",
			command:   "   ",
			wantError: true,
		},
		{
			name:      "dangerous rm -rf /",
			command:   "rm -rf /",
			wantError: true,
		},
		{
			name:      "dangerous rm -rf /*",
			command:   "rm -rf /*",
			wantError: true,
		},
		{
			name:      "safe rm command",
			command:   "rm -rf /tmp/test",
			wantError: false,
		},
		{
			name:      "dangerous mkfs",
			command:   "mkfs.ext4 /dev/sda",
			wantError: true,
		},
		{
			name:      "dangerous dd",
			command:   "dd if=/dev/zero of=/dev/sda",
			wantError: true,
		},
		{
			name:      "fork bomb",
			command:   ":(){ :|:& };:",
			wantError: true,
		},
		{
			name:      "dangerous chmod",
			command:   "chmod -R 777 /",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.command)
			if (err != nil) != tt.wantError {
				t.Errorf("validateCommand() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestRunCommand_ToolRegistration(t *testing.T) {
	tool, err := GetTool("run_command")
	if err != nil {
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
		tool, err = GetTool("run_command")
		if err != nil {
			t.Fatalf("Failed to get run_command tool after registration: %v", err)
		}
	}

	if tool.Name != "run_command" {
		t.Errorf("Expected tool name 'run_command', got: %s", tool.Name)
	}

	if tool.Function == nil {
		t.Errorf("Expected tool function to be set")
	}

	if tool.Description == "" {
		t.Errorf("Expected tool description to be set")
	}

	params := tool.Parameters

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected properties to be map[string]interface{}")
	}

	if _, ok := properties["command"]; !ok {
		t.Errorf("Expected 'command' property to be defined")
	}

	if _, ok := properties["working_directory"]; !ok {
		t.Errorf("Expected 'working_directory' property to be defined")
	}

	if _, ok := properties["timeout"]; !ok {
		t.Errorf("Expected 'timeout' property to be defined")
	}
}
