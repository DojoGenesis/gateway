// Package logging provides structured logging initialization for the Agentic Gateway.
//
// It configures the global slog.Default() logger based on the environment:
//   - production: JSON handler for machine parsing
//   - development: text handler for human readability
//
// Usage:
//
//	logging.Init("production")
//	slog.Info("server starting", "port", 7340)
package logging

import (
	"log/slog"
	"os"
)

// Init configures the global slog logger.
// environment should be "production" or "development".
func Init(environment string) {
	var handler slog.Handler

	if environment == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	}

	slog.SetDefault(slog.New(handler))
}
