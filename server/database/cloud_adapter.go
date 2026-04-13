package database

import (
	"context"
	"errors"
)

// ErrCloudAdapterNotImplemented is returned by all CloudAdapter methods.
// The cloud adapter (Supabase integration) is intentionally deferred — v1 is local-first,
// SQLite-only. This interface is preserved for future multi-user cloud deployment.
// The migration system handles this gracefully via errors.Is(err, ErrCloudAdapterNotImplemented).
var ErrCloudAdapterNotImplemented = errors.New("cloud adapter: intentionally deferred (v1 is local-first, SQLite-only)")

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

func (a *CloudAdapter) CreateMessage(ctx context.Context, msg *Message) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) ListMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetMessage(ctx context.Context, id string) (*Message, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) CreatePromptTemplate(ctx context.Context, tmpl *PromptTemplate) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetPromptTemplate(ctx context.Context, id string) (*PromptTemplate, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) ListPromptTemplates(ctx context.Context, userID string, includePublic bool) ([]*PromptTemplate, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) UpdatePromptTemplate(ctx context.Context, tmpl *PromptTemplate) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) DeletePromptTemplate(ctx context.Context, id string) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) CreateDocument(ctx context.Context, doc *Document) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetDocument(ctx context.Context, id string) (*Document, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) ListDocuments(ctx context.Context, userID string) ([]*Document, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) DeleteDocument(ctx context.Context, id string) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) UpdateDocumentStatus(ctx context.Context, id string, status string, chunkCount int) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) CreateDocumentChunks(ctx context.Context, chunks []*DocumentChunk) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) SearchDocumentChunks(ctx context.Context, userID string, query string, limit int) ([]*DocumentChunk, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) GetDocumentChunks(ctx context.Context, documentID string) ([]*DocumentChunk, error) {
	return nil, ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) Ping(ctx context.Context) error {
	return ErrCloudAdapterNotImplemented
}

func (a *CloudAdapter) Close() error {
	return nil
}
