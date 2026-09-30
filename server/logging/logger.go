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
	"strings"
)

// Init configures the global slog logger.
// environment should be "production" or "development".
func Init(environment string) {
	var handler slog.Handler

	// Same rule as middleware.IsProductionString (case- and
	// space-insensitive). Restated here because this leaf package must not
	// import middleware; the test pins that the two agree.
	if strings.EqualFold(strings.TrimSpace(environment), "production") {
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
