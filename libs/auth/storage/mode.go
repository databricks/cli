// Package storage selects and constructs the CLI's U2M token storage backend.
//
// The CLI is gaining an OS-native secure-storage mode behind an experimental
// opt-in (MS1 of the CLI GA rollout). A persistent plaintext mode ships
// separately (MS3). Until MS4, the default remains the legacy file-backed
// cache with dual-write host-keyed entries for older Go SDK versions.
//
// See documents/fy2027-q2/cli-ga/ for the rollout contract and project plan.
package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/env"
)

// StorageMode identifies which credential backend the CLI uses for U2M tokens.
type StorageMode string

const (
	// StorageModeLegacy is the pre-GA baseline. Writes to
	// ~/.databricks/token-cache.json with dual-write host-keyed entries for
	// older Go SDK versions (v0.61-v0.103).
	StorageModeLegacy StorageMode = "legacy"

	// StorageModeSecure writes tokens to the OS-native secure store
	// (macOS Keychain, Windows Credential Manager, Linux Secret Service).
	// Never dual-writes.
	StorageModeSecure StorageMode = "secure"

	// StorageModePlaintext is reserved for MS3. When enabled it will write
	// to ~/.databricks/token-cache.json without host-keyed entries.
	StorageModePlaintext StorageMode = "plaintext"
)

// EnvVar is the environment variable that selects the storage mode.
const EnvVar = "DATABRICKS_AUTH_STORAGE"

// ResolveStorageMode returns the storage mode to use for this invocation.
//
// Precedence:
//  1. override (typically from a command-level flag such as --secure-storage).
//  2. DATABRICKS_AUTH_STORAGE env var.
//  3. [__settings__].auth_storage in .databrickscfg.
//  4. StorageModeLegacy.
//
// An empty override means "no flag set; fall through to env/config/default."
func ResolveStorageMode(ctx context.Context, override StorageMode) (StorageMode, error) {
	if override != "" {
		if err := validateMode(override); err != nil {
			return "", err
		}
		return override, nil
	}

	if raw := env.Get(ctx, EnvVar); raw != "" {
		mode, err := parseMode(raw)
		if err != nil {
			return "", fmt.Errorf("%s: %w", EnvVar, err)
		}
		return mode, nil
	}

	configFilePath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	raw, err := databrickscfg.GetConfiguredAuthStorage(ctx, configFilePath)
	if err != nil {
		return "", fmt.Errorf("read auth_storage setting: %w", err)
	}
	if raw != "" {
		mode, err := parseMode(raw)
		if err != nil {
			return "", fmt.Errorf("auth_storage: %w", err)
		}
		return mode, nil
	}

	return StorageModeLegacy, nil
}

func parseMode(raw string) (StorageMode, error) {
	mode := StorageMode(strings.ToLower(strings.TrimSpace(raw)))
	if err := validateMode(mode); err != nil {
		return "", err
	}
	return mode, nil
}

func validateMode(m StorageMode) error {
	switch m {
	case StorageModeLegacy, StorageModeSecure, StorageModePlaintext:
		return nil
	default:
		return fmt.Errorf("unknown storage mode %q (want legacy, secure, or plaintext)", string(m))
	}
}
