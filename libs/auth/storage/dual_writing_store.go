package storage

import "context"

// TokenKeyProvider supplies the primary key used to store an OAuth token.
type TokenKeyProvider interface {
	GetCacheKey() string
}

type discoveredHostProvider interface {
	GetDiscoveredHost() string
}

type hostCacheKeyProvider interface {
	GetHostCacheKey() string
}

// DualWritingStore wraps a Store so every write under the primary OAuth cache
// key is also mirrored under the legacy host-based key.
type DualWritingStore struct {
	inner Store
	arg   TokenKeyProvider
}

// NewDualWritingStore returns a Store wrapping inner that mirrors
// writes made under arg.GetCacheKey() to the argument's host key when one
// can be derived (via DiscoveryOAuthArgument.GetDiscoveredHost or
// HostCacheKeyProvider.GetHostCacheKey).
func NewDualWritingStore(inner Store, arg TokenKeyProvider) *DualWritingStore {
	return &DualWritingStore{inner: inner, arg: arg}
}

// Lookup implements Store; delegates to the inner store.
func (s *DualWritingStore) Lookup(key string) (Entry, error) {
	return s.inner.Lookup(key)
}

// WithLock implements Store and keeps both mirrored mutations in one
// transaction on the inner store.
func (s *DualWritingStore) WithLock(ctx context.Context, fn func(LockedStore) error) error {
	return s.inner.WithLock(ctx, func(locked LockedStore) error {
		return fn(&dualLockedStore{store: s, inner: locked})
	})
}

type dualLockedStore struct {
	store *DualWritingStore
	inner LockedStore
}

// Lookup implements LockedStore.
func (s *dualLockedStore) Lookup(key string) (Entry, error) {
	return s.inner.Lookup(key)
}

// Put implements LockedStore. The host-key mirror remains best-effort for
// compatibility with older Go SDK versions.
func (s *dualLockedStore) Put(key string, e Entry) error {
	if err := s.inner.Put(key, e); err != nil {
		return err
	}
	hostKey := s.hostKey(key)
	if hostKey == "" {
		return nil
	}
	_ = s.inner.Put(hostKey, e)
	return nil
}

// Delete implements LockedStore and mirrors the deletion when key is primary.
func (s *dualLockedStore) Delete(key string) error {
	if err := s.inner.Delete(key); err != nil {
		return err
	}
	hostKey := s.hostKey(key)
	if hostKey == "" {
		return nil
	}
	_ = s.inner.Delete(hostKey)
	return nil
}

func (s *dualLockedStore) hostKey(key string) string {
	primaryKey := s.store.arg.GetCacheKey()
	if key != primaryKey {
		return ""
	}
	hostKey := hostCacheKey(s.store.arg)
	if hostKey == primaryKey {
		return ""
	}
	return hostKey
}

var _ Store = (*DualWritingStore)(nil)

// hostCacheKey mirrors the SDK's former PersistentAuth.hostCacheKey:
// discovery arguments expose the host via GetDiscoveredHost (populated by
// Challenge); static arguments expose it via HostCacheKeyProvider.
func hostCacheKey(arg TokenKeyProvider) string {
	if discoveryArg, ok := arg.(discoveredHostProvider); ok {
		return discoveryArg.GetDiscoveredHost()
	}
	if hcp, ok := arg.(hostCacheKeyProvider); ok {
		return hcp.GetHostCacheKey()
	}
	return ""
}
