package provider

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

const (
	PluginProtocolVersion = 1
	HandshakeKey          = "DOJO_GENESIS_PLUGIN"
	HandshakeValue        = "v0.0.15"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  PluginProtocolVersion,
	MagicCookieKey:   HandshakeKey,
	MagicCookieValue: HandshakeValue,
}

type PluginManager struct {
	config         PluginManagerConfig
	clients        map[string]*plugin.Client
	providers      map[string]ModelProvider
	pluginPaths    map[string]string
	pluginConfigs  map[string]map[string]interface{}
	restartCounts  map[string]int
	monitorCancels map[string]context.CancelFunc
	mu             sync.RWMutex
}

func NewPluginManager(pluginDir string) *PluginManager {
	return NewPluginManagerWithConfig(PluginManagerConfig{
		PluginDir:          pluginDir,
		MonitorInterval:    5 * time.Second,
		RestartDelay:       1 * time.Second,
		MaxRestartAttempts: 3,
	})
}

func NewPluginManagerWithConfig(config PluginManagerConfig) *PluginManager {
	if config.MonitorInterval == 0 {
		config.MonitorInterval = 5 * time.Second
	}
	if config.RestartDelay == 0 {
		config.RestartDelay = 1 * time.Second
	}
	if config.MaxRestartAttempts == 0 {
		config.MaxRestartAttempts = 3
	}

	return &PluginManager{
		config:         config,
		clients:        make(map[string]*plugin.Client),
		providers:      make(map[string]ModelProvider),
		pluginPaths:    make(map[string]string),
		pluginConfigs:  make(map[string]map[string]interface{}),
		restartCounts:  make(map[string]int),
		monitorCancels: make(map[string]context.CancelFunc),
	}
}

func (pm *PluginManager) DiscoverPlugins() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, err := os.Stat(pm.config.PluginDir); os.IsNotExist(err) {
		if err := os.MkdirAll(pm.config.PluginDir, 0755); err != nil {
			return fmt.Errorf("failed to create plugin directory: %w", err)
		}
	}

	// If provider configs are specified, only load those
	if len(pm.config.Providers) > 0 {
		for _, providerCfg := range pm.config.Providers {
			if !providerCfg.Enabled {
				slog.Info("skipping disabled plugin", "name", providerCfg.Name)
				continue
			}

			pluginPath := providerCfg.PluginPath
			if !filepath.IsAbs(pluginPath) {
				pluginPath = filepath.Join(pm.config.PluginDir, pluginPath)
			}

			info, err := os.Stat(pluginPath)
			if err != nil {
				slog.Error("failed to stat plugin", "name", providerCfg.Name, "error", err)
				continue
			}

			if info.Mode()&0111 == 0 {
				slog.Warn("plugin is not executable", "name", providerCfg.Name)
				continue
			}

			if err := pm.loadPluginWithConfig(providerCfg.Name, pluginPath, providerCfg.Config); err != nil {
				slog.Error("failed to load plugin", "name", providerCfg.Name, "error", err)
				continue
			}

			slog.Info("loaded plugin", "name", providerCfg.Name, "priority", providerCfg.Priority)
		}
		return nil
	}

	// Fallback: auto-discover all executable files in plugin directory
	entries, err := os.ReadDir(pm.config.PluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		pluginPath := filepath.Join(pm.config.PluginDir, name)

		info, err := os.Stat(pluginPath)
		if err != nil {
			slog.Error("failed to stat plugin", "name", name, "error", err)
			continue
		}

		if info.Mode()&0111 == 0 {
			continue
		}

		if err := pm.loadPluginWithConfig(name, pluginPath, nil); err != nil {
			slog.Error("failed to load plugin", "name", name, "error", err)
			continue
		}

		slog.Info("discovered and loaded plugin", "name", name)
	}

	return nil
}

func (pm *PluginManager) LoadPlugin(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pluginPath := filepath.Join(pm.config.PluginDir, name)
	return pm.loadPluginWithConfig(name, pluginPath, nil)
}

func (pm *PluginManager) loadPluginWithConfig(name, pluginPath string, config map[string]interface{}) error {
	if _, exists := pm.clients[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	cmd := exec.Command(pluginPath)
	cmd.Env = append(os.Environ(), buildConfigEnv(name, config)...)

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"provider": &ModelProviderGRPCPlugin{},
		},
		Cmd:              cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   fmt.Sprintf("plugin-%s", name),
			Output: os.Stdout,
			Level:  hclog.Info,
		}),
		Managed: true,
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to get RPC client: %w", err)
	}

	raw, err := rpcClient.Dispense("provider")
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to dispense plugin: %w", err)
	}

	provider, ok := raw.(ModelProvider)
	if !ok {
		client.Kill()
		return fmt.Errorf("plugin does not implement ModelProvider interface")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := provider.GetInfo(ctx)
	if err != nil {
		client.Kill()
		return fmt.Errorf("failed to get plugin info: %w", err)
	}

	if info.Version == "" {
		client.Kill()
		return fmt.Errorf("plugin missing version information")
	}

	pm.clients[name] = client
	pm.providers[name] = provider
	pm.pluginPaths[name] = pluginPath
	pm.pluginConfigs[name] = config
	pm.restartCounts[name] = 0

	if cancelFn, ok := pm.monitorCancels[name]; ok {
		cancelFn()
	}
	monCtx, monCancel := context.WithCancel(context.Background())
	pm.monitorCancels[name] = monCancel

	go pm.monitorPlugin(monCtx, name)

	return nil
}

func (pm *PluginManager) monitorPlugin(ctx context.Context, name string) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("plugin monitor cancelled", "name", name)
			return
		case <-time.After(pm.config.MonitorInterval):
		}

		pm.mu.RLock()
		client, exists := pm.clients[name]
		pm.mu.RUnlock()

		if !exists || !client.Exited() {
			continue
		}

		pm.mu.Lock()
		delete(pm.clients, name)
		delete(pm.providers, name)

		pluginPath, hasPath := pm.pluginPaths[name]
		pluginConfig := pm.pluginConfigs[name]
		restartCount := pm.restartCounts[name]
		maxAttempts := pm.config.MaxRestartAttempts
		pm.mu.Unlock()

		if !hasPath {
			slog.Error("plugin crashed but path not found, cannot restart", "name", name)
			return
		}

		if restartCount >= maxAttempts {
			slog.Error("plugin exceeded max restart attempts, giving up", "name", name, "max_attempts", maxAttempts)
			pm.mu.Lock()
			delete(pm.pluginPaths, name)
			delete(pm.pluginConfigs, name)
			delete(pm.restartCounts, name)
			delete(pm.monitorCancels, name)
			pm.mu.Unlock()
			return
		}

		slog.Warn("plugin crashed, restarting", "name", name, "attempt", restartCount+1, "max_attempts", maxAttempts)

		select {
		case <-ctx.Done():
			slog.Debug("plugin monitor cancelled during restart delay", "name", name)
			return
		case <-time.After(pm.config.RestartDelay):
		}

		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			slog.Error("plugin binary no longer exists, cannot restart", "name", name, "path", pluginPath)
			pm.mu.Lock()
			delete(pm.pluginPaths, name)
			delete(pm.pluginConfigs, name)
			delete(pm.restartCounts, name)
			delete(pm.monitorCancels, name)
			pm.mu.Unlock()
			return
		}

		pm.mu.Lock()
		pm.restartCounts[name] = restartCount + 1
		if err := pm.loadPluginWithConfig(name, pluginPath, pluginConfig); err != nil {
			slog.Error("failed to restart plugin", "name", name, "error", err)
			pm.mu.Unlock()
			return
		}
		slog.Info("successfully restarted plugin", "name", name)
		pm.mu.Unlock()
		return
	}
}

func (pm *PluginManager) GetProvider(name string) (ModelProvider, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	provider, exists := pm.providers[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return provider, nil
}

// RegisterProvider registers an in-process ModelProvider under the given name.
// This allows registering providers that don't require external plugin binaries.
func (pm *PluginManager) RegisterProvider(name string, provider ModelProvider) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.providers[name] = provider
}

func (pm *PluginManager) GetProviders() map[string]ModelProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	providers := make(map[string]ModelProvider, len(pm.providers))
	for name, provider := range pm.providers {
		providers[name] = provider
	}

	return providers
}

func (pm *PluginManager) Shutdown() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, cancel := range pm.monitorCancels {
		cancel()
		delete(pm.monitorCancels, name)
	}

	var errs []error

	for name, client := range pm.clients {
		slog.Info("shutting down plugin", "name", name)
		client.Kill()
		delete(pm.clients, name)
		delete(pm.providers, name)
		delete(pm.pluginPaths, name)
		delete(pm.pluginConfigs, name)
		delete(pm.restartCounts, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during shutdown: %v", errs)
	}

	return nil
}

func (pm *PluginManager) Cleanup() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	for name, cancel := range pm.monitorCancels {
		cancel()
		delete(pm.monitorCancels, name)
	}
}

// UpdatePluginConfig updates the config for a loaded plugin and restarts it.
// This is used when a user adds/changes an API key for a provider.
func (pm *PluginManager) UpdatePluginConfig(name string, configUpdates map[string]interface{}) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pluginPath, hasPath := pm.pluginPaths[name]
	if !hasPath {
		return fmt.Errorf("plugin %s not found or not loaded from disk", name)
	}

	// Merge config updates into existing config
	existingConfig := pm.pluginConfigs[name]
	mergedConfig := make(map[string]interface{})
	for k, v := range existingConfig {
		mergedConfig[k] = v
	}
	for k, v := range configUpdates {
		mergedConfig[k] = v
	}

	// Kill old plugin if running
	if client, exists := pm.clients[name]; exists {
		client.Kill()
	}
	delete(pm.clients, name)
	delete(pm.providers, name)
	pm.restartCounts[name] = 0

	// Reload with updated config
	if err := pm.loadPluginWithConfig(name, pluginPath, mergedConfig); err != nil {
		return fmt.Errorf("failed to restart plugin %s with updated config: %w", name, err)
	}

	slog.Info("plugin restarted with updated config", "name", name)
	return nil
}

// IsPluginLoaded returns whether a plugin with the given name is currently loaded.
func (pm *PluginManager) IsPluginLoaded(name string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, exists := pm.providers[name]
	return exists
}

// providerAPIKeyEnvVar returns the environment variable name that a given
// provider plugin expects for its API key.
func providerAPIKeyEnvVar(providerName string) string {
	switch providerName {
	case "deepseek-api":
		return "DEEPSEEK_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "google":
		return "GOOGLE_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "mistral":
		return "MISTRAL_API_KEY"
	default:
		return strings.ToUpper(strings.ReplaceAll(providerName, "-", "_")) + "_API_KEY"
	}
}

func buildConfigEnv(providerName string, config map[string]interface{}) []string {
	if config == nil {
		return nil
	}

	env := []string{}
	for key, value := range config {
		envKey := strings.ToUpper(key)
		envValue := fmt.Sprintf("%v", value)
		env = append(env, fmt.Sprintf("%s=%s", envKey, envValue))

		// Map generic "api_key" config to provider-specific env var
		if key == "api_key" {
			providerEnvKey := providerAPIKeyEnvVar(providerName)
			if providerEnvKey != "" {
				env = append(env, fmt.Sprintf("%s=%s", providerEnvKey, envValue))
			}
		}
	}
	return env
}
