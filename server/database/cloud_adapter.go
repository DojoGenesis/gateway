package database

import (
	"context"
	"errors"
)

var ErrCloudAdapterNotImplemented = errors.New("cloud adapter not yet implemented")

type CloudAdapter struct {
	supabaseURL string
	supabaseKey string
}

func NewCloudAdapter(supabaseURL, supabaseKey string) *CloudAdapter {
	return &CloudAdapter{
		supabaseURL: supabaseURL,
		supabaseKey: supabaseKey,
	}
}

func (a *CloudAdapter) GetUser(ctx context.Context, userID string) (*User, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) CreateUser(ctx context.Context, user *User) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) UpdateUser(ctx context.Context, user *User) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) StoreAPIKey(ctx context.Context, key *APIKey) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetAPIKey(ctx context.Context, userID, provider string) (*APIKey, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) ListAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) DeleteAPIKey(ctx context.Context, userID, provider string) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) UpdateAPIKeyLastUsed(ctx context.Context, userID, provider string) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) CreateConversation(ctx context.Context, conv *Conversation) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) ListConversations(ctx context.Context, userID string) ([]*Conversation, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) UpdateConversation(ctx context.Context, conv *Conversation) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) DeleteConversation(ctx context.Context, id string) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetSettings(ctx context.Context, userID string) (*Settings, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) CreateSettings(ctx context.Context, settings *Settings) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) UpdateSettings(ctx context.Context, settings *Settings) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) Ping(ctx context.Context) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) Close() error {
	return nil
}
