//go:build windows

package secure_storage

import (
	"context"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	advapi32       = syscall.NewLazyDLL("advapi32.dll")
	credWriteProc  = advapi32.NewProc("CredWriteW")
	credReadProc   = advapi32.NewProc("CredReadW")
	credDeleteProc = advapi32.NewProc("CredDeleteW")
	credFreeProc   = advapi32.NewProc("CredFree")
)

const (
	CRED_TYPE_GENERIC          = 1
	CRED_PERSIST_LOCAL_MACHINE = 2
	ERROR_NOT_FOUND            = 1168
)

type CREDENTIAL struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        syscall.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

type WindowsKeychain struct{}

func newPlatformKeychain() SecureStorage {
	return &WindowsKeychain{}
}

func (k *WindowsKeychain) Store(ctx context.Context, userID, provider, key string) error {
	if key == "" {
		return ErrInvalidKey
	}

	targetName := makeKeychainKey(userID, provider)
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return fmt.Errorf("failed to encode target name: %w", err)
	}

	userNamePtr, err := syscall.UTF16PtrFromString(ServiceName)
	if err != nil {
		return fmt.Errorf("failed to encode username: %w", err)
	}

	keyBytes := []byte(key)
	cred := CREDENTIAL{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetNamePtr,
		CredentialBlobSize: uint32(len(keyBytes)),
		CredentialBlob:     &keyBytes[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
		UserName:           userNamePtr,
	}

	ret, _, err := credWriteProc.Call(uintptr(unsafe.Pointer(&cred)), 0)
	if ret == 0 {
		return fmt.Errorf("%w: %v", ErrKeychainAccess, err)
	}

	return nil
}

func (k *WindowsKeychain) Retrieve(ctx context.Context, userID, provider string) (string, error) {
	targetName := makeKeychainKey(userID, provider)
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return "", fmt.Errorf("failed to encode target name: %w", err)
	}

	var credPtr uintptr
	ret, _, err := credReadProc.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		CRED_TYPE_GENERIC,
		0,
		uintptr(unsafe.Pointer(&credPtr)),
	)

	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno == ERROR_NOT_FOUND {
			return "", ErrKeyNotFound
		}
		return "", fmt.Errorf("%w: %v", ErrKeychainAccess, err)
	}

	defer credFreeProc.Call(credPtr)

	cred := (*CREDENTIAL)(unsafe.Pointer(credPtr))
	if cred.CredentialBlobSize == 0 {
		return "", ErrKeyNotFound
	}

	keyBytes := make([]byte, cred.CredentialBlobSize)
	for i := uint32(0); i < cred.CredentialBlobSize; i++ {
		keyBytes[i] = *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(cred.CredentialBlob)) + uintptr(i)))
	}

	return string(keyBytes), nil
}

func (k *WindowsKeychain) Delete(ctx context.Context, userID, provider string) error {
	targetName := makeKeychainKey(userID, provider)
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return fmt.Errorf("failed to encode target name: %w", err)
	}

	ret, _, err := credDeleteProc.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		CRED_TYPE_GENERIC,
		0,
	)

	if ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno == ERROR_NOT_FOUND {
			return ErrKeyNotFound
		}
		return fmt.Errorf("%w: %v", ErrKeychainAccess, err)
	}

	return nil
}

func (k *WindowsKeychain) IsAvailable(ctx context.Context) bool {
	return credWriteProc.Find() == nil && credReadProc.Find() == nil
}

func (k *WindowsKeychain) GetStorageType() string {
	return StorageTypeKeychain
}

func (k *WindowsKeychain) Close() error {
	return nil
}
