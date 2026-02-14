package secure_storage

import (
	"context"
	"errors"
)

var (
	ErrKeyNotFound        = errors.New("key not found in secure storage")
	ErrKeychainAccess     = errors.New("failed to access system keychain")
	ErrInvalidKey         = errors.New("invalid key format")
	ErrStorageUnavailable = errors.New("secure storage unavailable")
)

const (
	ServiceName         = "dojo-genesis"
	StorageTypeKeychain = "keychain"
	StorageTypeFallback = "fallback"
)

type SecureStorage interface {
	Store(ctx context.Context, userID, provider, key string) error
	Retrieve(ctx context.Context, userID, provider string) (string, error)
	Delete(ctx context.Context, userID, provider string) error
	IsAvailable(ctx context.Context) bool
	GetStorageType() string
	Close() error
}

func NewSecureStorage(fallbackDB string) (SecureStorage, error) {
	storage := newPlatformKeychain()

	if storage.IsAvailable(context.Background()) {
		return storage, nil
	}

	return NewFallbackStorage(fallbackDB)
}

func makeKeychainKey(userID, provider string) string {
	return "api_key:" + userID + ":" + provider
}
