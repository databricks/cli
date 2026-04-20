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

	// StorageModePlaintext is for backward compatibility and environments
	// that do not have access to an OS keyring. When enabled it will write
	// to ~/.databricks/token-cache.json without host-keyed entries.
	StorageModePlaintext StorageMode = "plaintext"
)

// EnvVar is the environment variable that selects the storage mode.
const EnvVar = "DATABRICKS_AUTH_STORAGE"

// ResolveStorageMode returns the storage mode to use for this invocation.
// It is a thin I/O wrapper: it reads the env var and the .databrickscfg
// setting, then delegates the precedence logic to resolveStorageMode.
//
// Precedence:
//  1. override (typically from a command-level flag such as --secure-storage).
//  2. DATABRICKS_AUTH_STORAGE env var.
//  3. [__settings__].auth_storage in .databrickscfg.
//  4. StorageModeLegacy.
//
// An empty override means "no flag set; fall through to env/config/default."
// An invalid override is a caller bug (callers are expected to pass typed
// StorageMode values); invalid env or config values are user-input errors
// and are wrapped with the source name for clarity.
func ResolveStorageMode(ctx context.Context, override StorageMode) (StorageMode, error) {
	envValue := env.Get(ctx, EnvVar)
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	configValue, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	if err != nil {
		return "", fmt.Errorf("read auth_storage setting: %w", err)
	}
	return resolveStorageMode(override, envValue, configValue)
}

// resolveStorageMode is the pure core of ResolveStorageMode. It takes the
// already-resolved values from the command-line override, the env var, and
// the config setting as plain strings, runs the precedence rules, and
// validates. No side effects, no I/O.
func resolveStorageMode(override StorageMode, envValue, configValue string) (StorageMode, error) {
	if override != "" {
		if err := validateMode(override); err != nil {
			return "", err
		}
		return override, nil
	}

	if envValue != "" {
		mode, err := parseMode(envValue)
		if err != nil {
			return "", fmt.Errorf("%s: %w", EnvVar, err)
		}
		return mode, nil
	}

	if configValue != "" {
		mode, err := parseMode(configValue)
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
