package storage

import (
	"context"
	"fmt"

	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
)

// cacheFactories bundles the constructors ResolveCache depends on. Extracted
// so unit tests can inject stubs without hitting the real OS keyring or
// filesystem. Production code uses defaultCacheFactories().
type cacheFactories struct {
	newFile    func() (cache.TokenCache, error)
	newKeyring func() cache.TokenCache
}

// defaultCacheFactories returns the production factory set.
// newFile is wrapped in a closure because cache.NewFileTokenCache is variadic
// (func(...FileTokenCacheOption)) and cannot satisfy the non-variadic field type
// by direct reference. The closure calls it with no options (SDK defaults).
func defaultCacheFactories() cacheFactories {
	return cacheFactories{
		newFile:    func() (cache.TokenCache, error) { return cache.NewFileTokenCache() },
		newKeyring: NewKeyringCache,
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
	case StorageModeLegacy, StorageModePlaintext:
		// Plaintext currently maps to the file cache; a dedicated
		// plaintext backend (no host-keyed dual-writes) is a follow-up.
		c, err := f.newFile()
		if err != nil {
			return nil, "", fmt.Errorf("open file token cache: %w", err)
		}
		return c, mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported storage mode %q", string(mode))
	}
}
