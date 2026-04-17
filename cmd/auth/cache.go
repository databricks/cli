package auth

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
)

// cacheFactories bundles the constructors newAuthCache depends on. Extracted
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
		newKeyring: storage.NewKeyringCache,
	}
}

// newAuthCache resolves the storage mode for this invocation and returns
// the corresponding token cache plus the resolved mode (so callers can log
// or surface it).
//
// override is usually the command-level flag value (e.g. the result of
// --secure-storage). Pass "" when the command has no flag.
func newAuthCache(ctx context.Context, override storage.StorageMode) (cache.TokenCache, storage.StorageMode, error) {
	return newAuthCacheWith(ctx, override, defaultCacheFactories())
}

// newAuthCacheWith is the pure form of newAuthCache. It takes the factory
// set as a parameter so tests can inject stubs.
func newAuthCacheWith(ctx context.Context, override storage.StorageMode, f cacheFactories) (cache.TokenCache, storage.StorageMode, error) {
	mode, err := storage.ResolveStorageMode(ctx, override)
	if err != nil {
		return nil, "", err
	}
	switch mode {
	case storage.StorageModeSecure:
		return f.newKeyring(), mode, nil
	case storage.StorageModeLegacy, storage.StorageModePlaintext:
		// MS1 ships no dedicated plaintext implementation; the switch will
		// be added in MS3. Until then the file cache is the safest existing
		// behavior and the resolver still accepts the value.
		c, err := f.newFile()
		if err != nil {
			return nil, "", fmt.Errorf("open file token cache: %w", err)
		}
		return c, mode, nil
	default:
		return nil, "", fmt.Errorf("unsupported storage mode %q", string(mode))
	}
}
