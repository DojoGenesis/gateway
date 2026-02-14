package examples

import (
	"context"
	"runtime"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/tools"
)

var startTime = time.Now()

// RegisterSystemTool registers a system information reference tool.
func RegisterSystemTool() error {
	return tools.RegisterTool(&tools.ToolDefinition{
		Name:        "get_system_info",
		Description: "Get system information including OS, architecture, Go version, and uptime",
		Parameters: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Function: getSystemInfoFunc,
	})
}

func getSystemInfoFunc(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"go_version":     runtime.Version(),
		"uptime_seconds": time.Since(startTime).Seconds(),
		"num_cpu":        runtime.NumCPU(),
		"num_goroutine":  runtime.NumGoroutine(),
	}, nil
}
