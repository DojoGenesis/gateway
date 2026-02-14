//go:build linux

package secure_storage

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type LinuxKeychain struct{}

func newPlatformKeychain() SecureStorage {
	return &LinuxKeychain{}
}

func (k *LinuxKeychain) Store(ctx context.Context, userID, provider, key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	account := makeKeychainKey(userID, provider)

	_ = k.Delete(ctx, userID, provider)

	cmd := exec.CommandContext(ctx, "secret-tool", "store",
		"--label", fmt.Sprintf("%s API Key", provider),
		"service", ServiceName,
		"account", account)

	cmd.Stdin = strings.NewReader(key)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrKeychainAccess, string(output), err)
	}

	return nil
}

func (k *LinuxKeychain) Retrieve(ctx context.Context, userID, provider string) (string, error) {
	account := makeKeychainKey(userID, provider)

	cmd := exec.CommandContext(ctx, "secret-tool", "lookup",
		"service", ServiceName,
		"account", account)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
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

func (k *LinuxKeychain) Delete(ctx context.Context, userID, provider string) error {
	account := makeKeychainKey(userID, provider)

	cmd := exec.CommandContext(ctx, "secret-tool", "clear",
		"service", ServiceName,
		"account", account)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return ErrKeyNotFound
			}
		}
		return fmt.Errorf("%w: %s: %v", ErrKeychainAccess, string(output), err)
	}

	return nil
}

func (k *LinuxKeychain) IsAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "secret-tool", "--version")
	err := cmd.Run()
	return err == nil
}

func (k *LinuxKeychain) GetStorageType() string {
	return StorageTypeKeychain
}

func (k *LinuxKeychain) Close() error {
	return nil
}
