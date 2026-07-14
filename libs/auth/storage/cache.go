package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
)

// storeFactories bundles the constructors ResolveStore depends on. Extracted
// so unit tests can inject stubs without hitting the real OS keyring or
// filesystem. Production code uses defaultStoreFactories().
type storeFactories struct {
	newFile          func(context.Context) (Store, error)
	newKeyring       func() Store
	probeKeyring     func() error
	probeKeyringRead func() error
}

// defaultStoreFactories returns the production factory set.
func defaultStoreFactories() storeFactories {
	return storeFactories{
		newFile:          func(ctx context.Context) (Store, error) { return NewFileStore(ctx) },
		newKeyring:       NewKeyringStore,
		probeKeyring:     ProbeKeyring,
		probeKeyringRead: ProbeKeyringRead,
	}
}

// ResolveStore resolves the storage mode for this invocation and returns
// the corresponding token cache plus the resolved mode (so callers can log
// or surface it).
//
// override is usually the command-level flag value. Pass "" when the command
// has no flag; precedence then falls through to env -> config -> default.
//
// When the resolver returns (mode=Secure, source=Default) and the OS
// keyring is definitively unreachable (a non-timeout probe error), reads
// fall back to the plaintext file cache so post-upgrade users with legacy
// token-cache.json entries are not stranded. Unlike the login path, this
// fallback does not persist auth_storage = plaintext to [__settings__];
// pinning happens only on successful login.
//
// Every CLI code path that calls u2m.NewPersistentAuth must route the result
// through u2m.WithTokenCache, otherwise the SDK defaults to the file cache
// and splits the user's tokens across two backends.
func ResolveStore(ctx context.Context, override StorageMode) (Store, StorageMode, error) {
	return resolveStoreForReadWith(ctx, override, defaultStoreFactories())
}

// ResolveStoreForLogin resolves the cache like ResolveStore with extra rules
// for the auth login path:
//
//  1. When the resolved mode is secure and the user did not explicitly ask
//     for it (no override flag, no env var, no config), and the OS keyring
//     is unreachable, fall back silently to plaintext and persist
//     auth_storage = plaintext to [__settings__] so subsequent commands
//     skip the (slow/blocking) probe and route directly to the file cache.
//  2. When the user explicitly asked for secure (override, env var, or
//     config) but the keyring is unreachable, return an error. An explicit
//     "I want secure" is honored strictly: never silently downgrade.
//  3. When the probe times out, stay on keyring regardless of explicit.
//     The timeout is ambiguous (locked vs hung); a misdiagnosis fails
//     the final Store rather than silently downgrading to plaintext.
//
// Login-specific. Read paths (auth token, bundle commands) keep the original
// keyring error so they don't silently mint plaintext copies of tokens that
// were stored in the keyring on another machine.
func ResolveStoreForLogin(ctx context.Context, override StorageMode) (Store, StorageMode, error) {
	return resolveStoreForLoginWith(ctx, override, defaultStoreFactories())
}

// OAuthTokenCache adapts a CLI Store to the SDK's u2m_cache.TokenCache for the
// U2M PersistentAuth flow, applying the not-found hint so a cache miss carries
// actionable "run databricks auth login" guidance. Use on read and credential
// paths. M2M/OIDC callers use the CLI Store directly and must not route through
// here: the login hint is the wrong remedy for those auth types.
func OAuthTokenCache(ctx context.Context, c Store, mode StorageMode) cache.TokenCache {
	return withNotFoundHint(ctx, ToU2MTokenCache(c), mode)
}

// WrapForOAuthArgument is OAuthTokenCache plus, in plaintext mode, a dual-write
// of every Challenge/refresh write to the legacy host-based cache key. Use on
// the login and refresh write paths. Other modes return the plain adapter:
// secure mode never writes a host-key entry, and the dual-write has nothing to
// do for non-file backends.
//
// Pass the OAuthArgument that the same NewPersistentAuth call will use. For
// discovery arguments the discovered host is read at Store time, so it is
// safe to wrap before Challenge populates it.
func WrapForOAuthArgument(ctx context.Context, c Store, mode StorageMode, arg u2m.OAuthArgument) cache.TokenCache {
	tc := OAuthTokenCache(ctx, c, mode)
	if mode != StorageModePlaintext {
		return tc
	}
	return NewDualWritingTokenCache(tc, arg)
}

// resolveStoreWith is the pure form of ResolveStore without the read-path
// fallback. Takes the factory set as a parameter so tests can inject stubs.
// Used directly by ResolveStoreForLogin (which has its own fallback rules)
// and indirectly by ResolveStore (which adds the read-path fallback in
// resolveStoreForReadWith).
func resolveStoreWith(ctx context.Context, override StorageMode, f storeFactories) (Store, StorageMode, error) {
	mode, err := ResolveStorageMode(ctx, override)
	if err != nil {
		return nil, "", err
	}
	switch mode {
	case StorageModeSecure:
		return f.newKeyring(), mode, nil
	case StorageModePlaintext:
		c, err := f.newFile(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("open file token cache: %w", err)
		}
		return c, mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported storage mode %q", string(mode))
	}
}

// resolveStoreForReadWith is the pure form of ResolveStore. It applies the
// read-path fallback: when mode is secure-from-default and the keyring
// probes as definitively unavailable, return the file cache instead.
// Timeouts keep the keyring (could be transient).
func resolveStoreForReadWith(ctx context.Context, override StorageMode, f storeFactories) (Store, StorageMode, error) {
	mode, source, err := ResolveStorageModeWithSource(ctx, override)
	if err != nil {
		return nil, "", err
	}
	return applyReadFallback(ctx, mode, source.Explicit(), f)
}

// applyReadFallback realizes the read-path fallback. Mirrors
// applyLoginFallback but:
//
//   - Uses a read-only probe (ProbeKeyringRead) so calls do not write to
//     the keyring on every CLI invocation.
//   - Does not persist auth_storage = plaintext. Pinning happens only on
//     successful login, where the write-probe gives us stronger evidence
//     that the keyring is truly unavailable on this machine.
//
// Explicit secure is honored: callers who asked for secure get the keyring
// cache even if the probe fails, so the actual Lookup error surfaces the
// unreachability instead of silently using a different backend.
func applyReadFallback(ctx context.Context, mode StorageMode, explicit bool, f storeFactories) (Store, StorageMode, error) {
	switch mode {
	case StorageModePlaintext:
		c, err := f.newFile(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("open file token cache: %w", err)
		}
		return c, mode, nil
	case StorageModeSecure:
		if explicit {
			return f.newKeyring(), mode, nil
		}
		if probeErr := f.probeKeyringRead(); probeErr != nil {
			if _, ok := errors.AsType[*TimeoutError](probeErr); ok {
				log.Debugf(ctx, "keyring read probe timed out (%v); staying on keyring", probeErr)
				return f.newKeyring(), mode, nil
			}
			log.Debugf(ctx, "secure storage unavailable on read path (%v), using file cache", probeErr)
			store, fileErr := f.newFile(ctx)
			if fileErr != nil {
				return nil, "", fmt.Errorf("open file token cache: %w", fileErr)
			}
			return store, StorageModePlaintext, nil
		}
		return f.newKeyring(), mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported storage mode %q", string(mode))
	}
}

// resolveStoreForLoginWith is the pure form of ResolveStoreForLogin. It takes
// the factory set as a parameter so tests can inject stubs.
func resolveStoreForLoginWith(ctx context.Context, override StorageMode, f storeFactories) (Store, StorageMode, error) {
	mode, source, err := ResolveStorageModeWithSource(ctx, override)
	if err != nil {
		return nil, "", err
	}
	return applyLoginFallback(ctx, mode, source.Explicit(), f)
}

// applyLoginFallback realizes the login-time fallback rules given an already-
// resolved mode and whether the user explicitly asked for it. Split out so
// tests can drive the (mode, explicit) input space directly without depending
// on whatever the resolver's default mode happens to be at any point in time.
func applyLoginFallback(ctx context.Context, mode StorageMode, explicit bool, f storeFactories) (Store, StorageMode, error) {
	switch mode {
	case StorageModePlaintext:
		c, err := f.newFile(ctx)
		if err != nil {
			return nil, "", fmt.Errorf("open file token cache: %w", err)
		}
		return c, mode, nil
	case StorageModeSecure:
		if probeErr := f.probeKeyring(); probeErr != nil {
			// Stay on keyring on timeout: a locked keyring being unlocked
			// during OAuth is the common case, and a misdiagnosed hang
			// fails the final Store anyway, which is better than a
			// silent plaintext downgrade.
			if _, ok := errors.AsType[*TimeoutError](probeErr); ok {
				log.Debugf(ctx, "keyring probe timed out (%v); staying on keyring", probeErr)
				return f.newKeyring(), mode, nil
			}
			if explicit {
				return nil, "", fmt.Errorf("secure storage was requested but the OS keyring is not reachable: %w", probeErr)
			}
			log.Debugf(ctx, "secure storage unavailable (%v), falling back to plaintext", probeErr)
			fileStore, fileErr := f.newFile(ctx)
			if fileErr != nil {
				return nil, "", fmt.Errorf("open file token cache: %w", fileErr)
			}
			persistPlaintextFallback(ctx)
			return fileStore, StorageModePlaintext, nil
		}
		return f.newKeyring(), mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported storage mode %q", string(mode))
	}
}

// persistPlaintextFallback writes auth_storage = plaintext to [__settings__]
// in .databrickscfg so subsequent commands skip the (slow/blocking) keyring
// probe and route straight to the file cache.
//
// Only called on the (mode=Secure, explicit=false) probe-failure branch.
// Best-effort: persistence failures are logged at debug and never block
// login.
func persistPlaintextFallback(ctx context.Context) {
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	if err := databrickscfg.SetConfiguredAuthStorage(ctx, string(StorageModePlaintext), configPath); err != nil {
		log.Debugf(ctx, "persist auth_storage=plaintext fallback failed: %v", err)
	}
}

// PinSecureMode persists auth_storage = secure to [__settings__] when the
// user is currently on the secure-from-default path. Once pinned, subsequent
// invocations see source=Config (explicit), so applyLoginFallback returns an
// error on a transient keyring probe failure instead of silently demoting
// the user to plaintext.
//
// No-op when mode is not secure or when the user already chose a mode
// explicitly via override, env var, or config. override must be the same
// value the caller passed to ResolveStoreForLogin so the source check sees
// the caller's intent rather than re-resolving without it.
//
// Persistence failures are logged at warn: they do not block login, but
// the user should know the pin did not happen, since a later transient
// keyring failure could then silently route a default-secure user to
// plaintext. Concurrent logins racing this write is benign because both
// write the same value.
func PinSecureMode(ctx context.Context, mode, override StorageMode) {
	if mode != StorageModeSecure {
		return
	}
	_, source, err := ResolveStorageModeWithSource(ctx, override)
	if err != nil {
		log.Debugf(ctx, "pin secure mode: resolve: %v", err)
		return
	}
	if source.Explicit() {
		return
	}
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	if err := databrickscfg.SetConfiguredAuthStorage(ctx, string(StorageModeSecure), configPath); err != nil {
		log.Warnf(ctx, "could not persist auth_storage=secure to %s: %v. Future commands may need DATABRICKS_AUTH_STORAGE=secure to keep using the OS keyring.", configPath, err)
	}
}
