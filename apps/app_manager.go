package apps

import (
	"context"
	"fmt"
	"time"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/pkg/gateway"
)

// AppManagerConfig holds configuration for the AppManager.
type AppManagerConfig struct {
	AllowedOrigins     []string
	DefaultToolTimeout time.Duration
	RequireUserConsent bool
}

// AppManagerStatus holds status information about the app manager.
type AppManagerStatus struct {
	ResourceCount  int  `json:"resource_count"`
	ActiveAppCount int  `json:"active_app_count"`
	Healthy        bool `json:"healthy"`
}

// AppManager is the top-level manager that wires all MCP Apps components together.
type AppManager struct {
	resources *ResourceRegistry
	apps      *AppRegistry
	proxy     *ToolCallProxy
	security  *SecurityPolicy
	authz     *AuthorizationPolicy
	config    AppManagerConfig
}

// NewAppManager creates an AppManager with all sub-components initialized.
func NewAppManager(cfg AppManagerConfig, toolRegistry gateway.ToolRegistry) *AppManager {
	resources := NewResourceRegistry()
	apps := NewAppRegistry()
	authz := NewAuthorizationPolicy()
	proxy := NewToolCallProxy(toolRegistry, authz)
	security := NewSecurityPolicy()

	if cfg.DefaultToolTimeout > 0 {
		proxy.SetDefaultTimeout(cfg.DefaultToolTimeout)
	}

	for _, origin := range cfg.AllowedOrigins {
		security.AddAllowedOrigin(origin)
	}

	return &AppManager{
		resources: resources,
		apps:      apps,
		proxy:     proxy,
		security:  security,
		authz:     authz,
		config:    cfg,
	}
}

// RegisterResource validates permissions and registers a resource.
func (m *AppManager) RegisterResource(meta *ResourceMeta) error {
	if err := m.security.ValidatePermissions(meta.Permissions); err != nil {
		return fmt.Errorf("resource %s has invalid permissions: %w", meta.URI, err)
	}
	return m.resources.Register(meta)
}

// LaunchApp validates the resource exists, launches an app instance, and grants tool access.
func (m *AppManager) LaunchApp(resourceURI, sessionID string) (*AppInstance, error) {
	if _, err := m.resources.Get(resourceURI); err != nil {
		return nil, fmt.Errorf("cannot launch app: %w", err)
	}

	inst, err := m.apps.Launch(resourceURI, sessionID)
	if err != nil {
		return nil, err
	}

	// Grant wildcard tool access to the launched app
	m.authz.GrantAllToolAccess(inst.ID)

	return inst, nil
}

// CloseApp closes an app instance and revokes its tool access.
func (m *AppManager) CloseApp(instanceID string) error {
	m.authz.RevokeAll(instanceID)
	return m.apps.Close(instanceID)
}

// ProxyToolCall delegates to the ToolCallProxy.
func (m *AppManager) ProxyToolCall(ctx context.Context, req *ToolCallRequest) (*ToolCallResponse, error) {
	return m.proxy.ProxyCall(ctx, req)
}

// GetResource retrieves a resource by URI.
func (m *AppManager) GetResource(uri string) (*ResourceMeta, error) {
	return m.resources.Get(uri)
}

// ListApps returns all app instances for a session.
func (m *AppManager) ListApps(sessionID string) []*AppInstance {
	return m.apps.ListBySession(sessionID)
}

// Status returns current counts and health information.
func (m *AppManager) Status() AppManagerStatus {
	return AppManagerStatus{
		ResourceCount:  len(m.resources.List()),
		ActiveAppCount: m.apps.Count(),
		Healthy:        true,
	}
}

// SecurityPolicy returns the security policy for header injection.
func (m *AppManager) SecurityPolicy() *SecurityPolicy {
	return m.security
}
