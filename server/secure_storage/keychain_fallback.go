package secure_storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TresPies-source/AgenticGatewayByDojoGenesis/server/database"
	_ "modernc.org/sqlite"
)

type FallbackStorage struct {
	db        *sql.DB
	aead      cipher.AEAD
	masterKey []byte
}

func NewFallbackStorage(dbPath string) (*FallbackStorage, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open fallback storage: %w", err)
	}

	if err := database.ConfigureSQLiteDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to configure fallback storage: %w", err)
	}

	if err := initFallbackDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize fallback storage: %w", err)
	}

	masterKey, err := getOrCreateMasterKey(db)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &FallbackStorage{
		db:        db,
		aead:      aead,
		masterKey: masterKey,
	}, nil
}

func initFallbackDB(db *sql.DB) error {
	schema := `
		CREATE TABLE IF NOT EXISTS secure_keys (
			user_id TEXT NOT NULL,
			provider TEXT NOT NULL,
			encrypted_value BLOB NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, provider)
		);

		CREATE TABLE IF NOT EXISTS master_key (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			key_hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(schema)
	return err
}

func getOrCreateMasterKey(db *sql.DB) ([]byte, error) {
	var keyHashStr string
	err := db.QueryRow("SELECT key_hash FROM master_key WHERE id = 1").Scan(&keyHashStr)

	if err == sql.ErrNoRows {
		masterKey := make([]byte, 32)
		if _, err := rand.Read(masterKey); err != nil {
			return nil, fmt.Errorf("failed to generate master key: %w", err)
		}

		saltBytes := make([]byte, 16)
		if _, err := rand.Read(saltBytes); err != nil {
			return nil, fmt.Errorf("failed to generate salt: %w", err)
		}

		hash := sha256.Sum256(masterKey)
		hashStr := base64.StdEncoding.EncodeToString(hash[:])
		saltStr := base64.StdEncoding.EncodeToString(saltBytes)

		keyEncoded := base64.StdEncoding.EncodeToString(masterKey)

		_, err = db.Exec("INSERT INTO master_key (id, key_hash, salt) VALUES (1, ?, ?)",
			keyEncoded, saltStr)
		if err != nil {
			return nil, fmt.Errorf("failed to store master key: %w", err)
		}

		_ = hashStr

		return masterKey, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query master key: %w", err)
	}

	masterKey, err := base64.StdEncoding.DecodeString(keyHashStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode master key: %w", err)
	}

	if len(masterKey) != 32 {
		return nil, fmt.Errorf("invalid master key length: %d", len(masterKey))
	}

	return masterKey, nil
}

func (f *FallbackStorage) encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, f.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := f.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return ciphertext, nil
}

func (f *FallbackStorage) decrypt(ciphertext []byte) (string, error) {
	if len(ciphertext) < f.aead.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:f.aead.NonceSize()]
	ciphertext = ciphertext[f.aead.NonceSize():]

	plaintext, err := f.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func (f *FallbackStorage) Store(ctx context.Context, userID, provider, key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	encrypted, err := f.encrypt(key)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	query := `
		INSERT INTO secure_keys (user_id, provider, encrypted_value, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, provider) DO UPDATE SET
			encrypted_value = excluded.encrypted_value,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err = f.db.ExecContext(ctx, query, userID, provider, encrypted)
	if err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}

	return nil
}

func (f *FallbackStorage) Retrieve(ctx context.Context, userID, provider string) (string, error) {
	var encrypted []byte
	query := "SELECT encrypted_value FROM secure_keys WHERE user_id = ? AND provider = ?"

	err := f.db.QueryRowContext(ctx, query, userID, provider).Scan(&encrypted)
	if err == sql.ErrNoRows {
		return "", ErrKeyNotFound
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve key: %w", err)
	}

	decrypted, err := f.decrypt(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key: %w", err)
	}

	return decrypted, nil
}

func (f *FallbackStorage) Delete(ctx context.Context, userID, provider string) error {
	query := "DELETE FROM secure_keys WHERE user_id = ? AND provider = ?"

	result, err := f.db.ExecContext(ctx, query, userID, provider)
	if err != nil {
		return fmt.Errorf("failed to delete key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return ErrKeyNotFound
	}

	return nil
}

func (f *FallbackStorage) IsAvailable(ctx context.Context) bool {
	return f.db.Ping() == nil
}

func (f *FallbackStorage) GetStorageType() string {
	return StorageTypeFallback
}

func (f *FallbackStorage) Close() error {
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}
