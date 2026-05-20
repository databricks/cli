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

// cacheFactories bundles the constructors ResolveCache depends on. Extracted
// so unit tests can inject stubs without hitting the real OS keyring or
// filesystem. Production code uses defaultCacheFactories().
type cacheFactories struct {
	newFile      func(context.Context) (cache.TokenCache, error)
	newKeyring   func() cache.TokenCache
	probeKeyring func() error
}

// defaultCacheFactories returns the production factory set.
func defaultCacheFactories() cacheFactories {
	return cacheFactories{
		newFile:      func(ctx context.Context) (cache.TokenCache, error) { return NewFileTokenCache(ctx) },
		newKeyring:   NewKeyringCache,
		probeKeyring: ProbeKeyring,
	}
}

// ResolveCache resolves the storage mode for this invocation and returns
// the corresponding token cache plus the resolved mode (so callers can log
// or surface it).
//
// override is usually the command-level flag value. Pass "" when the command
// has no flag; precedence then falls through to env -> config -> default.
//
// Every CLI code path that calls u2m.NewPersistentAuth must route the result
// through u2m.WithTokenCache, otherwise the SDK defaults to the file cache
// and splits the user's tokens across two backends.
func ResolveCache(ctx context.Context, override StorageMode) (cache.TokenCache, StorageMode, error) {
	inner, mode, err := resolveCacheWith(ctx, override, defaultCacheFactories())
	if err != nil {
		return nil, "", err
	}
	return withNotFoundHint(ctx, inner, mode), mode, nil
}

// ResolveCacheForLogin resolves the cache like ResolveCache with extra rules
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
func ResolveCacheForLogin(ctx context.Context, override StorageMode) (cache.TokenCache, StorageMode, error) {
	inner, mode, err := resolveCacheForLoginWith(ctx, override, defaultCacheFactories())
	if err != nil {
		return nil, "", err
	}
	return withNotFoundHint(ctx, inner, mode), mode, nil
}

// WrapForOAuthArgument wraps tokenCache so SDK-side writes (Challenge, refresh)
// dual-write to the legacy host-based cache key when mode is plaintext. Other
// modes return tokenCache unchanged: secure mode never writes a host-key entry,
// and the wrapper has nothing to do for non-file backends.
//
// Pass the OAuthArgument that the same NewPersistentAuth call will use. For
// discovery arguments the discovered host is read at Store time, so it is
// safe to wrap before Challenge populates it.
func WrapForOAuthArgument(tokenCache cache.TokenCache, mode StorageMode, arg u2m.OAuthArgument) cache.TokenCache {
	if mode != StorageModePlaintext {
		return tokenCache
	}
	return NewDualWritingTokenCache(tokenCache, arg)
}

// resolveCacheWith is the pure form of ResolveCache. It takes the factory
// set as a parameter so tests can inject stubs.
func resolveCacheWith(ctx context.Context, override StorageMode, f cacheFactories) (cache.TokenCache, StorageMode, error) {
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

// resolveCacheForLoginWith is the pure form of ResolveCacheForLogin. It takes
// the factory set as a parameter so tests can inject stubs.
func resolveCacheForLoginWith(ctx context.Context, override StorageMode, f cacheFactories) (cache.TokenCache, StorageMode, error) {
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
func applyLoginFallback(ctx context.Context, mode StorageMode, explicit bool, f cacheFactories) (cache.TokenCache, StorageMode, error) {
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
			var timeoutErr *TimeoutError
			if errors.As(probeErr, &timeoutErr) {
				log.Debugf(ctx, "keyring probe timed out (%v); staying on keyring", probeErr)
				return f.newKeyring(), mode, nil
			}
			if explicit {
				return nil, "", fmt.Errorf("secure storage was requested but the OS keyring is not reachable: %w", probeErr)
			}
			log.Debugf(ctx, "secure storage unavailable (%v), falling back to plaintext", probeErr)
			fileCache, fileErr := f.newFile(ctx)
			if fileErr != nil {
				return nil, "", fmt.Errorf("open file token cache: %w", fileErr)
			}
			persistPlaintextFallback(ctx)
			return fileCache, StorageModePlaintext, nil
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
// explicitly. Best-effort: persistence failures are logged at debug and
// never block login. Concurrent logins racing this write is benign because
// both write the same value.
func PinSecureMode(ctx context.Context, mode StorageMode) {
	if mode != StorageModeSecure {
		return
	}
	_, source, err := ResolveStorageModeWithSource(ctx, StorageModeUnknown)
	if err != nil {
		log.Debugf(ctx, "pin secure mode: resolve: %v", err)
		return
	}
	if source.Explicit() {
		return
	}
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	if err := databrickscfg.SetConfiguredAuthStorage(ctx, string(StorageModeSecure), configPath); err != nil {
		log.Debugf(ctx, "persist auth_storage=secure pin failed: %v", err)
	}
}
