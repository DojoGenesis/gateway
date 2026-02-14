package maintenance

import (
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/memory"
	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/provider"
)

// ProviderPluginManager is the interface that provider.PluginManager satisfies.
type ProviderPluginManager interface {
	GetProvider(name string) (provider.ModelProvider, error)
	GetProviders() map[string]provider.ModelProvider
}

// PluginManagerAdapter wraps a ProviderPluginManager to satisfy memory.PluginManagerInterface.
// This adapter exists because memory.PluginManagerInterface uses interface{} return types
// to avoid a circular dependency between memory and provider packages.
type PluginManagerAdapter struct {
	pm ProviderPluginManager
}

// NewPluginManagerAdapter creates a new adapter.
func NewPluginManagerAdapter(pm ProviderPluginManager) memory.PluginManagerInterface {
	return &PluginManagerAdapter{pm: pm}
}

func (a *PluginManagerAdapter) GetProvider(name string) (interface{}, error) {
	return a.pm.GetProvider(name)
}

func (a *PluginManagerAdapter) GetProviders() map[string]interface{} {
	providers := a.pm.GetProviders()
	result := make(map[string]interface{}, len(providers))
	for k, v := range providers {
		result[k] = v
	}
	return result
}
