package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DojoGenesis/gateway/provider"
	"github.com/DojoGenesis/gateway/server/config"
)

func TestRegisterProviders_NoKeys(t *testing.T) {
	// Clear all provider keys
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	// Disable Ollama detection
	savedOllama := os.Getenv("OLLAMA_HOST")
	os.Setenv("OLLAMA_HOST", "http://127.0.0.1:1")
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
		if savedOllama != "" {
			os.Setenv("OLLAMA_HOST", savedOllama)
		} else {
			os.Unsetenv("OLLAMA_HOST")
		}
	}()

	pm := provider.NewPluginManager("nonexistent-plugins")
	cfg := &config.Config{PluginDir: "nonexistent-plugins"}

	results := RegisterProviders(context.Background(), pm, cfg, nil)

	cloudLoaded := 0
	for _, r := range results {
		if r.Available {
			cloudLoaded++
		}
	}

	if cloudLoaded != 0 {
		t.Errorf("expected 0 providers loaded with no keys and no ollama, got %d", cloudLoaded)
	}
}

func TestRegisterProviders_SingleKey(t *testing.T) {
	// Clear all keys, set only ANTHROPIC_API_KEY
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	pm := provider.NewPluginManager("nonexistent-plugins")
	cfg := &config.Config{PluginDir: "nonexistent-plugins"}

	results := RegisterProviders(context.Background(), pm, cfg, nil)

	anthropicLoaded := false
	openaiLoaded := false
	for _, r := range results {
		if r.Name == "anthropic" && r.Available {
			anthropicLoaded = true
		}
		if r.Name == "openai" && r.Available {
			openaiLoaded = true
		}
	}

	if !anthropicLoaded {
		t.Error("expected anthropic to be loaded with ANTHROPIC_API_KEY set")
	}
	if openaiLoaded {
		t.Error("expected openai NOT to be loaded without OPENAI_API_KEY")
	}

	if !pm.IsPluginLoaded("anthropic") {
		t.Error("expected anthropic to be registered in PluginManager")
	}
}

func TestRegisterProviders_MultipleKeys(t *testing.T) {
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Setenv("OPENAI_API_KEY", "test-key")
	os.Setenv("GROQ_API_KEY", "test-key")

	pm := provider.NewPluginManager("nonexistent-plugins")
	cfg := &config.Config{PluginDir: "nonexistent-plugins"}

	results := RegisterProviders(context.Background(), pm, cfg, nil)

	loadedCount := 0
	for _, r := range results {
		if r.Available && r.Name != "ollama" {
			loadedCount++
		}
	}

	if loadedCount != 3 {
		t.Errorf("expected 3 cloud providers loaded, got %d", loadedCount)
	}
}

func TestRegisterProviders_OllamaAvailable(t *testing.T) {
	// Create a mock Ollama server
	ollamaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"models":[]}`))
	}))
	defer ollamaServer.Close()

	// Clear all keys
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	}()

	// Point Ollama at mock server
	os.Setenv("OLLAMA_HOST", ollamaServer.URL)
	defer os.Unsetenv("OLLAMA_HOST")

	pm := provider.NewPluginManager("nonexistent-plugins")
	cfg := &config.Config{PluginDir: "nonexistent-plugins"}

	results := RegisterProviders(context.Background(), pm, cfg, nil)

	ollamaLoaded := false
	for _, r := range results {
		if r.Name == "ollama" && r.Available {
			ollamaLoaded = true
		}
	}

	if !ollamaLoaded {
		t.Error("expected ollama to be loaded when server is available")
	}

	if !pm.IsPluginLoaded("ollama") {
		t.Error("expected ollama to be registered in PluginManager")
	}
}

func TestRegisterProviders_ProviderCount(t *testing.T) {
	keys := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GROQ_API_KEY", "MISTRAL_API_KEY", "DEEPSEEK_API_KEY", "KIMI_API_KEY"}
	saved := make(map[string]string)
	for _, k := range keys {
		saved[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	// Also disable Ollama detection for this test
	savedOllama := os.Getenv("OLLAMA_HOST")
	os.Setenv("OLLAMA_HOST", "http://127.0.0.1:1") // unreachable port
	defer func() {
		for k, v := range saved {
			if v != "" {
				os.Setenv(k, v)
			}
		}
		if savedOllama != "" {
			os.Setenv("OLLAMA_HOST", savedOllama)
		} else {
			os.Unsetenv("OLLAMA_HOST")
		}
	}()

	os.Setenv("ANTHROPIC_API_KEY", "test-key")
	os.Setenv("DEEPSEEK_API_KEY", "test-key")

	pm := provider.NewPluginManager("nonexistent-plugins")
	cfg := &config.Config{PluginDir: "nonexistent-plugins"}

	RegisterProviders(context.Background(), pm, cfg, nil)

	if pm.ProviderCount() != 2 {
		t.Errorf("expected ProviderCount 2, got %d", pm.ProviderCount())
	}
}
