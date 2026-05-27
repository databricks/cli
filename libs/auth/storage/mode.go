// Package storage selects and constructs the CLI's U2M token storage backend.
//
// Two modes are supported. Secure writes to the OS-native keyring under the
// profile cache key only; it is the resolver default. Plaintext writes to
// ~/.databricks/token-cache.json with host-key dual-write for older Go SDK
// versions (v0.61-v0.103); it is the opt-in fallback for environments where
// the OS keyring is not available.
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
	// StorageModeUnknown is the zero value. It means "no mode selected";
	// callers fall through to the next source in the precedence chain, or
	// to the default if no other source is set.
	StorageModeUnknown StorageMode = ""

	// StorageModePlaintext writes tokens to ~/.databricks/token-cache.json
	// and mirrors each token under the legacy host-based cache key for
	// older Go SDK versions (v0.61-v0.103). Opt-in via DATABRICKS_AUTH_STORAGE
	// or [__settings__].auth_storage for environments where the OS keyring
	// is not available.
	StorageModePlaintext StorageMode = "plaintext"

	// StorageModeSecure writes tokens to the OS-native secure store
	// (macOS Keychain, Windows Credential Manager, Linux Secret Service)
	// under the profile cache key only. No host-key entry is written.
	// This is the resolver default.
	StorageModeSecure StorageMode = "secure"
)

// EnvVar is the environment variable that selects the storage mode.
const EnvVar = "DATABRICKS_AUTH_STORAGE"

// StorageSource identifies which precedence level produced the resolved
// storage mode. Callers use it both to decide whether the user explicitly
// asked for a mode (everything except StorageSourceDefault) and to surface
// where the choice came from in user-facing output.
type StorageSource int

const (
	// StorageSourceDefault is the zero value: no override, env, or config
	// was set, so the resolver fell through to the built-in default.
	StorageSourceDefault StorageSource = iota
	StorageSourceOverride
	StorageSourceEnvVar
	StorageSourceConfig
)

// Explicit reports whether the source came from a user-supplied input
// (override flag, env var, or config) rather than the built-in default.
func (s StorageSource) Explicit() bool {
	return s != StorageSourceDefault
}

// String returns a human-readable label for the source, matching the style
// used by the SDK's config.Source.String() (e.g. "DATABRICKS_HOST environment
// variable").
//
// The label for StorageSourceConfig intentionally does not name a specific
// config file: callers that know the resolved path (e.g. auth describe)
// should append it themselves to match the SDK's "from <path> config file"
// convention. The label for StorageSourceOverride is generic because no
// CLI command currently exposes a storage-mode flag; if one is added in
// the future, that command can replace the label at the call site.
func (s StorageSource) String() string {
	switch s {
	case StorageSourceOverride:
		return "command-line override"
	case StorageSourceEnvVar:
		return EnvVar + " environment variable"
	case StorageSourceConfig:
		return "auth_storage in [__settings__] section of config file"
	default:
		return "default"
	}
}

// ParseMode parses raw as a StorageMode. Whitespace is trimmed and matching
// is case-insensitive. Empty or unrecognized input returns StorageModeUnknown;
// callers decide whether that is an error (user-supplied value) or a
// fall-through signal (absent input).
func ParseMode(raw string) StorageMode {
	switch m := StorageMode(strings.ToLower(strings.TrimSpace(raw))); m {
	case StorageModePlaintext, StorageModeSecure:
		return m
	default:
		return StorageModeUnknown
	}
}

// ResolveStorageMode returns the storage mode to use for this invocation.
//
// Precedence:
//  1. override (typically from a command-level flag such as --secure-storage).
//  2. DATABRICKS_AUTH_STORAGE env var.
//  3. [__settings__].auth_storage in .databrickscfg.
//  4. StorageModeSecure.
//
// StorageModeUnknown as override means "no flag set; fall through." The
// override is trusted to be a valid StorageMode: callers that parse user
// input into the type are responsible for validating at parse time. An
// unrecognized env or config value is reported as an error wrapped with
// the source name.
func ResolveStorageMode(ctx context.Context, override StorageMode) (StorageMode, error) {
	mode, _, err := ResolveStorageModeWithSource(ctx, override)
	return mode, err
}

// ResolveStorageModeWithSource is like ResolveStorageMode but also reports
// which precedence level produced the resolved mode. Callers use the source
// both to honor "I want secure" strictly (when source.Explicit() is true and
// secure cannot be provided, error out instead of silently downgrading) and
// to surface where the choice came from in user-facing output.
func ResolveStorageModeWithSource(ctx context.Context, override StorageMode) (StorageMode, StorageSource, error) {
	if override != StorageModeUnknown {
		return override, StorageSourceOverride, nil
	}

	if raw := env.Get(ctx, EnvVar); raw != "" {
		mode, err := parseFromSource(raw, EnvVar)
		return mode, StorageSourceEnvVar, err
	}

	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	raw, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	if err != nil {
		return "", StorageSourceDefault, fmt.Errorf("read auth_storage setting: %w", err)
	}
	if raw != "" {
		mode, err := parseFromSource(raw, "auth_storage")
		return mode, StorageSourceConfig, err
	}

	return StorageModeSecure, StorageSourceDefault, nil
}

func parseFromSource(raw, source string) (StorageMode, error) {
	mode := ParseMode(raw)
	if mode == StorageModeUnknown {
		return "", fmt.Errorf("%s: unknown storage mode %q (want plaintext or secure)", source, raw)
	}
	return mode, nil
}
