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
//  1. When the resolved mode is secure and the user did not explicitly ask for
//     it (no override flag, no env var, no config), and the OS keyring is
//     unreachable, fall back silently to plaintext and persist
//     auth_storage = plaintext so subsequent commands skip the probe.
//  2. When the user explicitly asked for secure (override, env var, or config)
//     but the keyring is unreachable, return an error. An explicit "I want
//     secure" is honored strictly: never silently downgrade.
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
	if mode != StorageModeSecure {
		switch mode {
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
	if probeErr := f.probeKeyring(); probeErr != nil {
		if explicit {
			return nil, "", fmt.Errorf("secure storage was requested but the OS keyring is not reachable: %w", probeErr)
		}
		log.Debugf(ctx, "secure storage unavailable (%v), falling back to plaintext", probeErr)
		fileCache, fileErr := f.newFile(ctx)
		if fileErr != nil {
			return nil, "", fmt.Errorf("open file token cache: %w", fileErr)
		}
		if err := persistPlaintextFallback(ctx); err != nil {
			log.Debugf(ctx, "persisting auth_storage fallback failed: %v", err)
		}
		return fileCache, StorageModePlaintext, nil
	}
	return f.newKeyring(), StorageModeSecure, nil
}

// persistPlaintextFallback writes auth_storage = plaintext to [__settings__]
// in .databrickscfg so subsequent commands skip the (slow/blocking) keyring
// probe and route straight to the file cache.
//
// We deliberately persist only on the default-mode + probe-fail path, never
// on the success paths:
//   - default + probe ok: writing the runtime mode would lock the current
//     default into the user's config and prevent a future change to the
//     default from reaching them.
//   - explicit secure (override, env, config): the value is already set
//     somewhere by definition, so a write would be redundant.
//
// The fallback is the only path where persisting changes future behavior.
// It also pins these users to plaintext explicitly, so any future changes to
// this logic don't accidentally disrupt them: they're already using plaintext
// implicitly (the keyring is unreachable), and the persisted setting makes
// that choice stable across CLI versions.
func persistPlaintextFallback(ctx context.Context) error {
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	return databrickscfg.SetConfiguredAuthStorage(ctx, string(StorageModePlaintext), configPath)
}
