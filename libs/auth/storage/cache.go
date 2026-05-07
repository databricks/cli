package storage

import (
	"context"
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
	return resolveCacheWith(ctx, override, defaultCacheFactories())
}

// ResolveCacheForLogin resolves the cache like ResolveCache with extra rules
// for the auth login path:
//
//  1. After a successful resolution, pin the resolved mode by writing
//     auth_storage to [__settings__] if the key is not already set. This
//     locks in the first working behavior (secure or plaintext) so a
//     subsequent invocation skips the keyring probe and doesn't oscillate
//     between modes if the keyring becomes flaky.
//  2. When the resolved mode is secure and the user did not explicitly ask
//     for it (no override flag, no env var, no config), and the OS keyring
//     is unreachable, fall back silently to plaintext.
//  3. When the user explicitly asked for secure (override, env var, or
//     config) but the keyring is unreachable, return an error. An explicit
//     "I want secure" is honored strictly: never silently downgrade.
//
// Login-specific. Read paths (auth token, bundle commands) keep the original
// keyring error so they don't silently mint plaintext copies of tokens that
// were stored in the keyring on another machine.
func ResolveCacheForLogin(ctx context.Context, override StorageMode) (cache.TokenCache, StorageMode, error) {
	return resolveCacheForLoginWith(ctx, override, defaultCacheFactories())
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
	mode, explicit, err := ResolveStorageModeWithSource(ctx, override)
	if err != nil {
		return nil, "", err
	}
	return applyLoginFallback(ctx, mode, explicit, f)
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
		pinResolvedMode(ctx, mode)
		return c, mode, nil
	case StorageModeSecure:
		if probeErr := f.probeKeyring(); probeErr != nil {
			if explicit {
				return nil, "", fmt.Errorf("secure storage was requested but the OS keyring is not reachable: %w", probeErr)
			}
			log.Debugf(ctx, "secure storage unavailable (%v), falling back to plaintext", probeErr)
			fileCache, fileErr := f.newFile(ctx)
			if fileErr != nil {
				return nil, "", fmt.Errorf("open file token cache: %w", fileErr)
			}
			pinResolvedMode(ctx, StorageModePlaintext)
			return fileCache, StorageModePlaintext, nil
		}
		pinResolvedMode(ctx, mode)
		return f.newKeyring(), mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported storage mode %q", string(mode))
	}
}

// pinResolvedMode writes auth_storage = mode to [__settings__] in
// .databrickscfg only if the key is not already set. The first successful
// login pins whichever mode worked; later logins with a different transient
// source (override flag, env var) do not overwrite the user's pinned
// preference. Once pinned, ResolveStorageModeWithSource reads the value as
// "explicit" and the resolver routes straight to the chosen backend, which
// also makes the keyring probe redundant for subsequent secure logins.
//
// Best-effort: a write failure is logged at debug and not returned. Users
// have already authenticated successfully by the time we get here, and the
// only consequence of a missing pin is a redundant probe (or fallback) on
// the next login.
func pinResolvedMode(ctx context.Context, mode StorageMode) {
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	existing, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	if err != nil {
		log.Debugf(ctx, "reading existing auth_storage failed: %v", err)
		return
	}
	if existing != "" {
		return
	}
	if err := databrickscfg.SetConfiguredAuthStorage(ctx, string(mode), configPath); err != nil {
		log.Debugf(ctx, "persisting auth_storage = %s failed: %v", mode, err)
	}
}
