// Package storage selects and constructs the CLI's U2M token storage backend.
//
// Two modes are supported. Plaintext writes to ~/.databricks/token-cache.json
// with host-key dual-write for older Go SDK versions (v0.61-v0.103); it is the
// resolver default. Secure writes to the OS-native keyring under the profile
// cache key only; it is opt-in pre-GA and slated to become the default at GA.
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
	// older Go SDK versions (v0.61-v0.103). This is the resolver default.
	StorageModePlaintext StorageMode = "plaintext"

	// StorageModeSecure writes tokens to the OS-native secure store
	// (macOS Keychain, Windows Credential Manager, Linux Secret Service)
	// under the profile cache key only. No host-key entry is written.
	StorageModeSecure StorageMode = "secure"
)

// EnvVar is the environment variable that selects the storage mode.
const EnvVar = "DATABRICKS_AUTH_STORAGE"

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
//  4. StorageModePlaintext.
//
// StorageModeUnknown as override means "no flag set; fall through." The
// override is trusted to be a valid StorageMode: callers that parse user
// input into the type are responsible for validating at parse time. An
// unrecognized env or config value is reported as an error wrapped with
// the source name.
func ResolveStorageMode(ctx context.Context, override StorageMode) (StorageMode, error) {
	if override != StorageModeUnknown {
		return override, nil
	}

	if raw := env.Get(ctx, EnvVar); raw != "" {
		return parseFromSource(raw, EnvVar)
	}

	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	raw, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	if err != nil {
		return "", fmt.Errorf("read auth_storage setting: %w", err)
	}
	if raw != "" {
		return parseFromSource(raw, "auth_storage")
	}

	return StorageModePlaintext, nil
}

func parseFromSource(raw, source string) (StorageMode, error) {
	mode := ParseMode(raw)
	if mode == StorageModeUnknown {
		return "", fmt.Errorf("%s: unknown storage mode %q (want plaintext or secure)", source, raw)
	}
	return mode, nil
}
