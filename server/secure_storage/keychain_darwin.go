//go:build darwin

package secure_storage

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type DarwinKeychain struct{}

func newPlatformKeychain() SecureStorage {
	return &DarwinKeychain{}
}

func (k *DarwinKeychain) Store(ctx context.Context, userID, provider, key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	account := makeKeychainKey(userID, provider)

	deleteCmd := exec.CommandContext(ctx, "security", "delete-generic-password",
		"-s", ServiceName,
		"-a", account)
	_ = deleteCmd.Run()

	addCmd := exec.CommandContext(ctx, "security", "add-generic-password",
		"-s", ServiceName,
		"-a", account,
		"-w", key,
		"-U")

	output, err := addCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrKeychainAccess, string(output), err)
	}

	return nil
}

func (k *DarwinKeychain) Retrieve(ctx context.Context, userID, provider string) (string, error) {
	account := makeKeychainKey(userID, provider)

	cmd := exec.CommandContext(ctx, "security", "find-generic-password",
		"-s", ServiceName,
		"-a", account,
		"-w")

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 44 {
				return "", ErrKeyNotFound
			}
		}
		return "", fmt.Errorf("%w: %v", ErrKeychainAccess, err)
	}

	key := strings.TrimSpace(string(output))
	if key == "" {
		return "", ErrKeyNotFound
	}

	return key, nil
}

func (k *DarwinKeychain) Delete(ctx context.Context, userID, provider string) error {
	account := makeKeychainKey(userID, provider)

	cmd := exec.CommandContext(ctx, "security", "delete-generic-password",
		"-s", ServiceName,
		"-a", account)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 44 {
				return ErrKeyNotFound
			}
		}
		return fmt.Errorf("%w: %s: %v", ErrKeychainAccess, string(output), err)
	}

	return nil
}

func (k *DarwinKeychain) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "security", "-h")
	err := cmd.Run()
	return err == nil
}

func (k *DarwinKeychain) GetStorageType() string {
	return StorageTypeKeychain
}

func (k *DarwinKeychain) Close() error {
	return nil
}
