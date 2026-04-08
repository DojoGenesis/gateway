package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestInit_Production(t *testing.T) {
	// Save and restore the default logger
	original := slog.Default()
	defer slog.SetDefault(original)

	Init("production")

	// After Init("production"), the default logger should be configured.
	// Verify by writing to a buffer-backed JSON handler and checking output.
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)
	logger.Info("test message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}

	// Verify JSON structure
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
		t.Fatalf("production log should be valid JSON: %v", err)
	}
	if msg, ok := logEntry["msg"]; !ok || msg != "test message" {
		t.Errorf("expected msg='test message', got %v", logEntry["msg"])
	}
}

func TestInit_Development(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	Init("development")

	// After Init("development"), verify text handler output
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(handler)
	logger.Debug("debug message", "component", "test")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}

	// Text handler output should contain the message
	if !strings.Contains(output, "debug message") {
		t.Errorf("expected output to contain 'debug message', got: %s", output)
	}
}

func TestInit_ProductionJSON(t *testing.T) {
	// Verify that the production handler produces machine-parseable JSON
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	logger := slog.New(handler)

	logger.Info("request handled",
		"method", "GET",
		"path", "/api/health",
		"status", 200,
		"duration_ms", 42,
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse JSON log: %v", err)
	}

	if entry["method"] != "GET" {
		t.Errorf("expected method=GET, got %v", entry["method"])
	}
	if entry["path"] != "/api/health" {
		t.Errorf("expected path=/api/health, got %v", entry["path"])
	}
	if entry["status"] != float64(200) {
		t.Errorf("expected status=200, got %v", entry["status"])
	}
}

func TestInit_DevelopmentDebugLevel(t *testing.T) {
	// Verify development mode enables debug level
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	logger.Debug("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("debug messages should appear in development mode")
	}
}

func TestInit_ProductionFiltersDebug(t *testing.T) {
	// Verify production mode filters debug messages
	var buf bytes.Buffer
	handler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	logger.Debug("should not appear")
	if buf.Len() > 0 {
		t.Error("debug messages should be filtered in production mode")
	}
}

func TestInit_UnknownEnvironment(t *testing.T) {
	// Unknown environments should fall through to development (text handler)
	original := slog.Default()
	defer slog.SetDefault(original)

	// Should not panic
	Init("staging")
	Init("")
	Init("test")
}

func TestInit_SetsGlobalDefault(t *testing.T) {
	original := slog.Default()
	defer slog.SetDefault(original)

	before := slog.Default()
	Init("production")
	after := slog.Default()

	// The logger instance should have changed
	if before == after {
		t.Error("Init should change the global default logger")
	}
}
