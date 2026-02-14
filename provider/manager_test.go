package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	info          *ProviderInfo
	models        []ModelInfo
	shouldCrash   bool
	getInfoError  error
	listModelsErr error
}

func (m *mockProvider) GetInfo(ctx context.Context) (*ProviderInfo, error) {
	if m.getInfoError != nil {
		return nil, m.getInfoError
	}
	if m.shouldCrash {
		os.Exit(1)
	}
	return m.info, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if m.listModelsErr != nil {
		return nil, m.listModelsErr
	}
	return m.models, nil
}

func (m *mockProvider) GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{
		ID:      "test-completion",
		Model:   req.Model,
		Content: "This is a mock response",
		Usage: Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	}, nil
}

func (m *mockProvider) GenerateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan *CompletionChunk, error) {
	ch := make(chan *CompletionChunk, 3)
	ch <- &CompletionChunk{ID: "1", Delta: "Hello", Done: false}
	ch <- &CompletionChunk{ID: "2", Delta: " World", Done: false}
	ch <- &CompletionChunk{ID: "3", Delta: "", Done: true}
	close(ch)
	return ch, nil
}

func (m *mockProvider) CallTool(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	return &ToolCallResponse{
		Result: "tool result",
		Error:  "",
	}, nil
}

func (m *mockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(i) / 1000.0
	}
	return embedding, nil
}

func createMockPlugin(t *testing.T, pluginDir, name string, provider *mockProvider) string {
	pluginPath := filepath.Join(pluginDir, name)

	pluginCode := fmt.Sprintf(`package main

import (
	"context"
	"fmt"
	
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	goplugin "github.com/hashicorp/go-plugin"
)

type mockProvider struct{}

func (m *mockProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:         %q,
		Version:      "1.0.0",
		Description:  "Mock provider for testing",
		Capabilities: []string{"completion", "streaming"},
	}, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return []provider.ModelInfo{
		{
			ID:          "mock-model-1",
			Name:        "Mock Model 1",
			Provider:    %q,
			ContextSize: 4096,
			Cost:        0.0,
		},
	}, nil
}

func (m *mockProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return &provider.CompletionResponse{
		ID:      "test-completion",
		Model:   req.Model,
		Content: "This is a mock response",
		Usage: provider.Usage{
			InputTokens:  10,
			OutputTokens: 20,
			TotalTokens:  30,
		},
	}, nil
}

func (m *mockProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	ch := make(chan *provider.CompletionChunk, 1)
	close(ch)
	return ch, nil
}

func (m *mockProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return &provider.ToolCallResponse{
		Result: "tool result",
		Error:  "",
	}, nil
}

func (m *mockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(i) / 1000.0
	}
	return embedding, nil
}

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: provider.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"provider": &provider.ModelProviderGRPCPlugin{Impl: &mockProvider{}},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
`, name, name)

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "main.go")

	err := os.WriteFile(srcFile, []byte(pluginCode), 0644)
	require.NoError(t, err)

	cwd, err := os.Getwd()
	require.NoError(t, err)

	cmd := exec.Command("go", "build", "-o", pluginPath, srcFile)
	cmd.Dir = filepath.Dir(cwd)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", output)
		require.NoError(t, err, "Failed to build mock plugin")
	}

	err = os.Chmod(pluginPath, 0755)
	require.NoError(t, err)

	return pluginPath
}

func TestNewPluginManager(t *testing.T) {
	pm := NewPluginManager("/tmp/test-plugins")
	assert.NotNil(t, pm)
	assert.Equal(t, "/tmp/test-plugins", pm.config.PluginDir)
	assert.NotNil(t, pm.clients)
	assert.NotNil(t, pm.providers)
	assert.Equal(t, 5*time.Second, pm.config.MonitorInterval)
	assert.Equal(t, 1*time.Second, pm.config.RestartDelay)
	assert.Equal(t, 3, pm.config.MaxRestartAttempts)
}

func TestPluginManager_DiscoverPlugins(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "mock-plugin-1", &mockProvider{
		info: &ProviderInfo{
			Name:    "mock-plugin-1",
			Version: "1.0.0",
		},
	})
	createMockPlugin(t, pluginDir, "mock-plugin-2", &mockProvider{
		info: &ProviderInfo{
			Name:    "mock-plugin-2",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.DiscoverPlugins()
	require.NoError(t, err)

	providers := pm.GetProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "mock-plugin-1")
	assert.Contains(t, providers, "mock-plugin-2")

	pm.Shutdown()
}

func TestPluginManager_DiscoverPluginsCreatesDirectory(t *testing.T) {
	pluginDir := filepath.Join(t.TempDir(), "plugins")

	pm := NewPluginManager(pluginDir)
	err := pm.DiscoverPlugins()
	require.NoError(t, err)

	_, err = os.Stat(pluginDir)
	assert.NoError(t, err, "Plugin directory should be created")

	pm.Shutdown()
}

func TestPluginManager_LoadPlugin(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	provider, err := pm.GetProvider("test-plugin")
	require.NoError(t, err)
	assert.NotNil(t, provider)

	ctx := context.Background()
	info, err := provider.GetInfo(ctx)
	require.NoError(t, err)
	assert.Equal(t, "test-plugin", info.Name)
	assert.Equal(t, "1.0.0", info.Version)

	pm.Shutdown()
}

func TestPluginManager_GetProvider(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	provider, err := pm.GetProvider("test-plugin")
	require.NoError(t, err)
	assert.NotNil(t, provider)

	_, err = pm.GetProvider("non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	pm.Shutdown()
}

func TestPluginManager_GetProviders(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "plugin-1", &mockProvider{
		info: &ProviderInfo{
			Name:    "plugin-1",
			Version: "1.0.0",
		},
	})
	createMockPlugin(t, pluginDir, "plugin-2", &mockProvider{
		info: &ProviderInfo{
			Name:    "plugin-2",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.DiscoverPlugins()
	require.NoError(t, err)

	providers := pm.GetProviders()
	assert.Len(t, providers, 2)
	assert.Contains(t, providers, "plugin-1")
	assert.Contains(t, providers, "plugin-2")

	pm.Shutdown()
}

func TestPluginManager_Shutdown(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	err = pm.Shutdown()
	assert.NoError(t, err)

	providers := pm.GetProviders()
	assert.Len(t, providers, 0, "All providers should be removed after shutdown")
}

func TestPluginManager_ProviderFunctionality(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
		models: []ModelInfo{
			{ID: "model-1", Name: "Model 1"},
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	provider, err := pm.GetProvider("test-plugin")
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("GetInfo", func(t *testing.T) {
		info, err := provider.GetInfo(ctx)
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", info.Name)
		assert.Equal(t, "1.0.0", info.Version)
	})

	t.Run("ListModels", func(t *testing.T) {
		models, err := provider.ListModels(ctx)
		require.NoError(t, err)
		assert.Len(t, models, 1)
		assert.Equal(t, "mock-model-1", models[0].ID)
	})

	t.Run("GenerateCompletion", func(t *testing.T) {
		resp, err := provider.GenerateCompletion(ctx, &CompletionRequest{
			Model: "mock-model-1",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "This is a mock response", resp.Content)
		assert.Equal(t, 30, resp.Usage.TotalTokens)
	})

	t.Run("CallTool", func(t *testing.T) {
		resp, err := provider.CallTool(ctx, &ToolCallRequest{
			ToolCall: ToolCall{
				Name: "test-tool",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "tool result", resp.Result)
	})

	pm.Shutdown()
}

func TestPluginManager_LoadPluginTwice(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	err = pm.LoadPlugin("test-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already loaded")

	pm.Shutdown()
}

func TestPluginManager_NonExecutableFile(t *testing.T) {
	pluginDir := t.TempDir()

	nonExecFile := filepath.Join(pluginDir, "not-executable.txt")
	err := os.WriteFile(nonExecFile, []byte("test"), 0644)
	require.NoError(t, err)

	pm := NewPluginManager(pluginDir)
	err = pm.DiscoverPlugins()
	require.NoError(t, err)

	providers := pm.GetProviders()
	assert.Len(t, providers, 0, "Non-executable files should be ignored")

	pm.Shutdown()
}

func TestPluginManager_ConcurrentAccess(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			provider, err := pm.GetProvider("test-plugin")
			assert.NoError(t, err)
			assert.NotNil(t, provider)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	pm.Shutdown()
}

func TestPluginManager_PluginVersionValidation(t *testing.T) {
	pluginDir := t.TempDir()

	pluginCode := `package main

import (
	"context"
	"fmt"
	
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
	goplugin "github.com/hashicorp/go-plugin"
)

type mockProvider struct{}

func (m *mockProvider) GetInfo(ctx context.Context) (*provider.ProviderInfo, error) {
	return &provider.ProviderInfo{
		Name:         "version-test",
		Version:      "",
		Description:  "Missing version",
		Capabilities: []string{},
	}, nil
}

func (m *mockProvider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return nil, nil
}

func (m *mockProvider) GenerateCompletion(ctx context.Context, req *provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return nil, nil
}

func (m *mockProvider) GenerateCompletionStream(ctx context.Context, req *provider.CompletionRequest) (<-chan *provider.CompletionChunk, error) {
	return nil, nil
}

func (m *mockProvider) CallTool(ctx context.Context, req *provider.ToolCallRequest) (*provider.ToolCallResponse, error) {
	return nil, nil
}

func (m *mockProvider) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = float32(i) / 1000.0
	}
	return embedding, nil
}

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: provider.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"provider": &provider.ModelProviderGRPCPlugin{Impl: &mockProvider{}},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
	})
}
`

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "main.go")
	pluginPath := filepath.Join(pluginDir, "no-version-plugin")

	err := os.WriteFile(srcFile, []byte(pluginCode), 0644)
	require.NoError(t, err)

	cwd, err := os.Getwd()
	require.NoError(t, err)

	cmd := exec.Command("go", "build", "-o", pluginPath, srcFile)
	cmd.Dir = filepath.Dir(cwd)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", output)
	}
	require.NoError(t, err)

	err = os.Chmod(pluginPath, 0755)
	require.NoError(t, err)

	pm := NewPluginManager(pluginDir)
	err = pm.LoadPlugin("no-version-plugin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing version")

	pm.Shutdown()
}

func TestPluginManager_DiscoverPluginsWithSubdirectories(t *testing.T) {
	pluginDir := t.TempDir()

	subDir := filepath.Join(pluginDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	createMockPlugin(t, pluginDir, "plugin-1", &mockProvider{
		info: &ProviderInfo{
			Name:    "plugin-1",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err = pm.DiscoverPlugins()
	require.NoError(t, err)

	providers := pm.GetProviders()
	assert.Len(t, providers, 1, "Should ignore subdirectories")

	pm.Shutdown()
}

func TestPluginManager_GetProviderNotFound(t *testing.T) {
	pm := NewPluginManager(t.TempDir())

	provider, err := pm.GetProvider("non-existent")
	assert.Error(t, err)
	assert.Nil(t, provider)
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManager_ShutdownEmpty(t *testing.T) {
	pm := NewPluginManager(t.TempDir())
	err := pm.Shutdown()
	assert.NoError(t, err)
}

func TestPluginManager_LoadNonExistentPlugin(t *testing.T) {
	pluginDir := t.TempDir()
	pm := NewPluginManager(pluginDir)

	err := pm.LoadPlugin("non-existent-plugin")
	assert.Error(t, err)
}

func TestPluginManager_StreamingCompletion(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "stream-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "stream-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("stream-plugin")
	require.NoError(t, err)

	provider, err := pm.GetProvider("stream-plugin")
	require.NoError(t, err)

	ctx := context.Background()
	ch, err := provider.GenerateCompletionStream(ctx, &CompletionRequest{
		Model: "test-model",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})
	require.NoError(t, err)

	chunks := []string{}
	for chunk := range ch {
		chunks = append(chunks, chunk.Delta)
	}

	assert.Len(t, chunks, 0, "Mock streaming returns empty channel")

	pm.Shutdown()
}

func TestPluginManager_ProviderAfterShutdown(t *testing.T) {
	pluginDir := t.TempDir()

	createMockPlugin(t, pluginDir, "test-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "test-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("test-plugin")
	require.NoError(t, err)

	err = pm.Shutdown()
	require.NoError(t, err)

	_, err = pm.GetProvider("test-plugin")
	assert.Error(t, err, "Should not find provider after shutdown")
}

func TestPluginManager_MultipleShutdowns(t *testing.T) {
	pm := NewPluginManager(t.TempDir())

	err := pm.Shutdown()
	assert.NoError(t, err)

	err = pm.Shutdown()
	assert.NoError(t, err, "Multiple shutdowns should not error")
}

func TestPluginManager_ProviderCount(t *testing.T) {
	pm := NewPluginManager(t.TempDir())

	assert.Equal(t, 0, pm.ProviderCount())

	pm.RegisterProvider("provider-1", &mockProvider{
		info: &ProviderInfo{Name: "provider-1", Version: "1.0.0"},
	})
	assert.Equal(t, 1, pm.ProviderCount())

	pm.RegisterProvider("provider-2", &mockProvider{
		info: &ProviderInfo{Name: "provider-2", Version: "1.0.0"},
	})
	assert.Equal(t, 2, pm.ProviderCount())
}

func TestPluginManager_ProviderStatuses(t *testing.T) {
	pm := NewPluginManager(t.TempDir())

	// Empty
	statuses := pm.ProviderStatuses()
	assert.Len(t, statuses, 0)

	// Register healthy provider
	pm.RegisterProvider("healthy", &mockProvider{
		info: &ProviderInfo{Name: "healthy", Version: "1.0.0"},
	})

	// Register unhealthy provider (GetInfo returns error)
	pm.RegisterProvider("unhealthy", &mockProvider{
		getInfoError: fmt.Errorf("provider down"),
	})

	statuses = pm.ProviderStatuses()
	assert.Len(t, statuses, 2)
	assert.True(t, statuses["healthy"])
	assert.False(t, statuses["unhealthy"])
}

func TestPluginManager_IsPluginLoaded(t *testing.T) {
	pm := NewPluginManager(t.TempDir())

	assert.False(t, pm.IsPluginLoaded("test-provider"))

	pm.RegisterProvider("test-provider", &mockProvider{
		info: &ProviderInfo{Name: "test-provider", Version: "1.0.0"},
	})

	assert.True(t, pm.IsPluginLoaded("test-provider"))
	assert.False(t, pm.IsPluginLoaded("other-provider"))
}

func BenchmarkPluginManager_GetProvider(b *testing.B) {
	pluginDir := b.TempDir()

	t := &testing.T{}
	createMockPlugin(t, pluginDir, "bench-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "bench-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("bench-plugin")
	if err != nil {
		b.Fatal(err)
	}

	defer pm.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pm.GetProvider("bench-plugin")
	}
}

func BenchmarkPluginManager_ProviderCall(b *testing.B) {
	pluginDir := b.TempDir()

	t := &testing.T{}
	createMockPlugin(t, pluginDir, "bench-plugin", &mockProvider{
		info: &ProviderInfo{
			Name:    "bench-plugin",
			Version: "1.0.0",
		},
	})

	pm := NewPluginManager(pluginDir)
	err := pm.LoadPlugin("bench-plugin")
	if err != nil {
		b.Fatal(err)
	}

	defer pm.Shutdown()

	provider, err := pm.GetProvider("bench-plugin")
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.GetInfo(ctx)
	}
}
