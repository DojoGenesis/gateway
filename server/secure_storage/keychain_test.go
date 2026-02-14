package secure_storage

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFallbackStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	testSecureStorage(t, storage)
}

func TestPlatformKeychain(t *testing.T) {
	storage := newPlatformKeychain()

	if !storage.IsAvailable(context.Background()) {
		t.Skip("Platform keychain not available on this system")
	}

	testSecureStorage(t, storage)
}

func TestNewSecureStorage(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewSecureStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	assert.True(t, storage.IsAvailable(context.Background()))

	storageType := storage.GetStorageType()
	assert.Contains(t, []string{StorageTypeKeychain, StorageTypeFallback}, storageType)
}

func testSecureStorage(t *testing.T, storage SecureStorage) {
	ctx := context.Background()

	t.Run("Store and Retrieve", func(t *testing.T) {
		userID := "test-user-1"
		provider := "openai"
		key := "sk-test-key-12345"

		err := storage.Store(ctx, userID, provider, key)
		require.NoError(t, err)

		retrieved, err := storage.Retrieve(ctx, userID, provider)
		require.NoError(t, err)
		assert.Equal(t, key, retrieved)
	})

	t.Run("Update Existing Key", func(t *testing.T) {
		userID := "test-user-2"
		provider := "anthropic"
		key1 := "sk-ant-old-key"
		key2 := "sk-ant-new-key"

		err := storage.Store(ctx, userID, provider, key1)
		require.NoError(t, err)

		err = storage.Store(ctx, userID, provider, key2)
		require.NoError(t, err)

		retrieved, err := storage.Retrieve(ctx, userID, provider)
		require.NoError(t, err)
		assert.Equal(t, key2, retrieved)
	})

	t.Run("Delete Key", func(t *testing.T) {
		userID := "test-user-3"
		provider := "deepseek"
		key := "sk-deepseek-key"

		err := storage.Store(ctx, userID, provider, key)
		require.NoError(t, err)

		err = storage.Delete(ctx, userID, provider)
		require.NoError(t, err)

		_, err = storage.Retrieve(ctx, userID, provider)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("Retrieve Non-existent Key", func(t *testing.T) {
		userID := "test-user-4"
		provider := "non-existent"

		_, err := storage.Retrieve(ctx, userID, provider)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("Delete Non-existent Key", func(t *testing.T) {
		userID := "test-user-5"
		provider := "non-existent"

		err := storage.Delete(ctx, userID, provider)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("Store Empty Key", func(t *testing.T) {
		userID := "test-user-6"
		provider := "openai"
		key := ""

		err := storage.Store(ctx, userID, provider, key)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("Multiple Users Same Provider", func(t *testing.T) {
		user1 := "test-user-7"
		user2 := "test-user-8"
		provider := "openai"
		key1 := "sk-user1-key"
		key2 := "sk-user2-key"

		err := storage.Store(ctx, user1, provider, key1)
		require.NoError(t, err)

		err = storage.Store(ctx, user2, provider, key2)
		require.NoError(t, err)

		retrieved1, err := storage.Retrieve(ctx, user1, provider)
		require.NoError(t, err)
		assert.Equal(t, key1, retrieved1)

		retrieved2, err := storage.Retrieve(ctx, user2, provider)
		require.NoError(t, err)
		assert.Equal(t, key2, retrieved2)
	})

	t.Run("Multiple Providers Same User", func(t *testing.T) {
		userID := "test-user-9"
		provider1 := "openai"
		provider2 := "anthropic"
		key1 := "sk-openai-key"
		key2 := "sk-anthropic-key"

		err := storage.Store(ctx, userID, provider1, key1)
		require.NoError(t, err)

		err = storage.Store(ctx, userID, provider2, key2)
		require.NoError(t, err)

		retrieved1, err := storage.Retrieve(ctx, userID, provider1)
		require.NoError(t, err)
		assert.Equal(t, key1, retrieved1)

		retrieved2, err := storage.Retrieve(ctx, userID, provider2)
		require.NoError(t, err)
		assert.Equal(t, key2, retrieved2)
	})

	t.Run("IsAvailable", func(t *testing.T) {
		assert.True(t, storage.IsAvailable(ctx))
	})

	t.Run("GetStorageType", func(t *testing.T) {
		storageType := storage.GetStorageType()
		assert.Contains(t, []string{StorageTypeKeychain, StorageTypeFallback}, storageType)
	})
}

func TestFallbackStorageEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"
	key := "sk-test-key-sensitive-data"

	err = storage.Store(ctx, userID, provider, key)
	require.NoError(t, err)

	storage.Close()

	rawDB, err := os.ReadFile(dbPath)
	require.NoError(t, err)

	dbContent := string(rawDB)
	assert.NotContains(t, dbContent, key, "Key should be encrypted in database")

	storage2, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage2.Close()

	retrieved, err := storage2.Retrieve(ctx, userID, provider)
	require.NoError(t, err)
	assert.Equal(t, key, retrieved)
}

func TestMakeKeychainKey(t *testing.T) {
	tests := []struct {
		userID   string
		provider string
		expected string
	}{
		{"user1", "openai", "api_key:user1:openai"},
		{"user2", "anthropic", "api_key:user2:anthropic"},
		{"guest-123", "deepseek", "api_key:guest-123:deepseek"},
	}

	for _, tt := range tests {
		t.Run(tt.userID+"-"+tt.provider, func(t *testing.T) {
			result := makeKeychainKey(tt.userID, tt.provider)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFallbackStorageClose(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)

	err = storage.Close()
	assert.NoError(t, err)

	err = storage.Close()
	assert.NoError(t, err)
}

func TestPlatformKeychainAvailability(t *testing.T) {
	storage := newPlatformKeychain()
	ctx := context.Background()

	isAvailable := storage.IsAvailable(ctx)

	switch runtime.GOOS {
	case "darwin":
		assert.True(t, isAvailable, "macOS should have keychain available")
	case "windows":
		assert.True(t, isAvailable, "Windows should have credential manager available")
	case "linux":
		if _, err := os.Stat("/usr/bin/secret-tool"); err == nil {
			assert.True(t, isAvailable, "Linux with secret-tool should be available")
		}
	}
}

func TestContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	userID := "test-user"
	provider := "openai"
	key := "sk-test-key"

	err = storage.Store(ctx, userID, provider, key)
	assert.Error(t, err)
}

func TestLongKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"

	longKey := make([]byte, 10000)
	for i := range longKey {
		longKey[i] = byte('a' + (i % 26))
	}
	key := string(longKey)

	err = storage.Store(ctx, userID, provider, key)
	require.NoError(t, err)

	retrieved, err := storage.Retrieve(ctx, userID, provider)
	require.NoError(t, err)
	assert.Equal(t, key, retrieved)
}

func TestSpecialCharactersInKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"
	key := "sk-test-!@#$%^&*()_+-={}[]|\\:;\"'<>,.?/"

	err = storage.Store(ctx, userID, provider, key)
	require.NoError(t, err)

	retrieved, err := storage.Retrieve(ctx, userID, provider)
	require.NoError(t, err)
	assert.Equal(t, key, retrieved)
}

func BenchmarkFallbackStore(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(b, err)
	defer storage.Close()

	ctx := context.Background()
	userID := "bench-user"
	provider := "openai"
	key := "sk-bench-key-12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.Store(ctx, userID, provider, key)
	}
}

func BenchmarkFallbackRetrieve(b *testing.B) {
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(b, err)
	defer storage.Close()

	ctx := context.Background()
	userID := "bench-user"
	provider := "openai"
	key := "sk-bench-key-12345"

	err = storage.Store(ctx, userID, provider, key)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Retrieve(ctx, userID, provider)
	}
}

func TestFallbackStorageDBErrors(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)

	storage.Close()

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"

	err = storage.Store(ctx, userID, provider, "key")
	assert.Error(t, err)

	_, err = storage.Retrieve(ctx, userID, provider)
	assert.Error(t, err)

	err = storage.Delete(ctx, userID, provider)
	assert.Error(t, err)

	assert.False(t, storage.IsAvailable(ctx))
}

func TestFallbackStorageInvalidDir(t *testing.T) {
	dbPath := "/invalid/nonexistent/directory/test.db"

	_, err := NewFallbackStorage(dbPath)
	assert.Error(t, err)
}

func TestFallbackEncryptionErrors(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewFallbackStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	shortCiphertext := []byte("short")
	_, err = storage.decrypt(shortCiphertext)
	assert.Error(t, err)

	invalidCiphertext := make([]byte, storage.aead.NonceSize()+16)
	for i := range invalidCiphertext {
		invalidCiphertext[i] = byte(i)
	}
	_, err = storage.decrypt(invalidCiphertext)
	assert.Error(t, err)
}

func TestPlatformKeychainClose(t *testing.T) {
	storage := newPlatformKeychain()
	err := storage.Close()
	assert.NoError(t, err)
}

func TestSecureStorageFallback(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_secure.db")

	storage, err := NewSecureStorage(dbPath)
	require.NoError(t, err)
	defer storage.Close()

	ctx := context.Background()
	userID := "test-user"
	provider := "openai"
	key := "sk-test-key"

	err = storage.Store(ctx, userID, provider, key)
	require.NoError(t, err)

	retrieved, err := storage.Retrieve(ctx, userID, provider)
	require.NoError(t, err)
	assert.Equal(t, key, retrieved)
}
