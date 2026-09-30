package logging

import (
	"log/slog"
	"testing"
)

// Init chose JSON/Info for exactly "production" and text/Debug with source
// locations for anything else, so a host set to "Production" logged at debug
// level in production.
func TestInit_ProductionIsCaseAndSpaceInsensitive(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	for _, env := range []string{"production", "Production", " PRODUCTION "} {
		Init(env)
		if _, ok := slog.Default().Handler().(*slog.JSONHandler); !ok {
			t.Errorf("Init(%q): handler %T, want *slog.JSONHandler", env, slog.Default().Handler())
		}
	}
	Init("development")
	if _, ok := slog.Default().Handler().(*slog.TextHandler); !ok {
		t.Errorf("Init(development): handler %T, want *slog.TextHandler", slog.Default().Handler())
	}
}
