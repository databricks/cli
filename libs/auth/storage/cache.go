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

// ResolveCacheForLogin resolves the cache like ResolveCache, but if the
// resolved mode is secure and the OS keyring is unreachable it silently
// falls back to plaintext and persists auth_storage = plaintext to
// .databrickscfg (only if no value is set there yet).
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
	tokenCache, mode, err := resolveCacheWith(ctx, override, f)
	if err != nil {
		return nil, "", err
	}
	if mode != StorageModeSecure {
		return tokenCache, mode, nil
	}
	if probeErr := f.probeKeyring(); probeErr != nil {
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
	return tokenCache, mode, nil
}

// persistPlaintextFallback writes auth_storage = plaintext to [__settings__]
// in .databrickscfg, but only if the key is not already set. This makes the
// silent fallback sticky across commands: future invocations resolve directly
// to plaintext and skip the keyring probe.
func persistPlaintextFallback(ctx context.Context) error {
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	existing, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	if err != nil {
		return err
	}
	if existing != "" {
		return nil
	}
	return databrickscfg.SetConfiguredAuthStorage(ctx, string(StorageModePlaintext), configPath)
}
