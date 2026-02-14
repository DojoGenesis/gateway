package database

import (
	"context"
	"errors"
)

var (
	ErrNoUserContext     = errors.New("no user context found")
	ErrInvalidAdapter    = errors.New("invalid database adapter")
	ErrAdapterNotEnabled = errors.New("adapter not enabled")
)

type UserContext struct {
	UserID   string
	UserType UserType
	Token    string
}

type DatabaseManager struct {
	localAdapter DatabaseAdapter
	cloudAdapter DatabaseAdapter
	useCloud     bool
}

func NewDatabaseManager(localAdapter DatabaseAdapter, cloudAdapter DatabaseAdapter, useCloud bool) *DatabaseManager {
	return &DatabaseManager{
		localAdapter: localAdapter,
		cloudAdapter: cloudAdapter,
		useCloud:     useCloud,
	}
}

func (m *DatabaseManager) getAdapter(userType UserType) (DatabaseAdapter, error) {
	if userType == UserTypeGuest {
		if m.localAdapter == nil {
			return nil, ErrInvalidAdapter
		}
		return m.localAdapter, nil
	}

	if userType == UserTypeAuthenticated {
		if m.useCloud && m.cloudAdapter != nil {
			return m.cloudAdapter, nil
		}
		if m.localAdapter == nil {
			return nil, ErrInvalidAdapter
		}
		return m.localAdapter, nil
	}

	return nil, ErrInvalidUserType
}

func (m *DatabaseManager) GetUser(ctx context.Context, userID string, userType UserType) (*User, error) {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return nil, err
	}
	return adapter.GetUser(ctx, userID)
}

func (m *DatabaseManager) CreateUser(ctx context.Context, user *User) error {
	adapter, err := m.getAdapter(user.UserType)
	if err != nil {
		return err
	}
	return adapter.CreateUser(ctx, user)
}

func (m *DatabaseManager) UpdateUser(ctx context.Context, user *User) error {
	adapter, err := m.getAdapter(user.UserType)
	if err != nil {
		return err
	}
	return adapter.UpdateUser(ctx, user)
}

func (m *DatabaseManager) StoreAPIKey(ctx context.Context, key *APIKey, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.StoreAPIKey(ctx, key)
}

func (m *DatabaseManager) GetAPIKey(ctx context.Context, userID, provider string, userType UserType) (*APIKey, error) {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return nil, err
	}
	return adapter.GetAPIKey(ctx, userID, provider)
}

func (m *DatabaseManager) ListAPIKeys(ctx context.Context, userID string, userType UserType) ([]*APIKey, error) {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return nil, err
	}
	return adapter.ListAPIKeys(ctx, userID)
}

// FindAPIKeyByProvider finds any active API key for a given provider across all users.
// Used by the in-process provider system to resolve API keys added via the Dev Mode UI.
// Tries local adapter first (where guest keys are stored), then cloud adapter.
func (m *DatabaseManager) FindAPIKeyByProvider(ctx context.Context, provider string) (*APIKey, error) {
	// Try local adapter first (most likely for guest users / Dev Mode setup)
	if m.localAdapter != nil {
		if la, ok := m.localAdapter.(*LocalAdapter); ok {
			key, err := la.FindAPIKeyByProvider(ctx, provider)
			if err == nil {
				return key, nil
			}
		}
	}

	// Try cloud adapter as fallback
	if m.cloudAdapter != nil {
		if la, ok := m.cloudAdapter.(*LocalAdapter); ok {
			key, err := la.FindAPIKeyByProvider(ctx, provider)
			if err == nil {
				return key, nil
			}
		}
	}

	return nil, ErrAPIKeyNotFound
}

// ListAllActiveAPIKeys returns all active API keys across all users.
// Used during bootstrap to configure plugins with stored keys.
func (m *DatabaseManager) ListAllActiveAPIKeys(ctx context.Context) ([]*APIKey, error) {
	if m.localAdapter != nil {
		if la, ok := m.localAdapter.(*LocalAdapter); ok {
			return la.ListAllActiveAPIKeys(ctx)
		}
	}
	return nil, nil
}

func (m *DatabaseManager) DeleteAPIKey(ctx context.Context, userID, provider string, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.DeleteAPIKey(ctx, userID, provider)
}

func (m *DatabaseManager) UpdateAPIKeyLastUsed(ctx context.Context, userID, provider string, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.UpdateAPIKeyLastUsed(ctx, userID, provider)
}

func (m *DatabaseManager) CreateConversation(ctx context.Context, conv *Conversation, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.CreateConversation(ctx, conv)
}

func (m *DatabaseManager) GetConversation(ctx context.Context, id string, userType UserType) (*Conversation, error) {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return nil, err
	}
	return adapter.GetConversation(ctx, id)
}

func (m *DatabaseManager) ListConversations(ctx context.Context, userID string, userType UserType) ([]*Conversation, error) {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return nil, err
	}
	return adapter.ListConversations(ctx, userID)
}

func (m *DatabaseManager) UpdateConversation(ctx context.Context, conv *Conversation, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.UpdateConversation(ctx, conv)
}

func (m *DatabaseManager) DeleteConversation(ctx context.Context, id string, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.DeleteConversation(ctx, id)
}

func (m *DatabaseManager) GetSettings(ctx context.Context, userID string, userType UserType) (*Settings, error) {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return nil, err
	}
	return adapter.GetSettings(ctx, userID)
}

func (m *DatabaseManager) CreateSettings(ctx context.Context, settings *Settings, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.CreateSettings(ctx, settings)
}

func (m *DatabaseManager) UpdateSettings(ctx context.Context, settings *Settings, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.UpdateSettings(ctx, settings)
}

func (m *DatabaseManager) Ping(ctx context.Context, userType UserType) error {
	adapter, err := m.getAdapter(userType)
	if err != nil {
		return err
	}
	return adapter.Ping(ctx)
}

func (m *DatabaseManager) Close() error {
	var errs []error

	if m.localAdapter != nil {
		if err := m.localAdapter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if m.cloudAdapter != nil {
		if err := m.cloudAdapter.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
