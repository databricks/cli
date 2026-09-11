package storage

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

// DualWritingStore wraps a Store so that every write under the
// primary OAuth cache key is also mirrored under the legacy host-based key.
// This preserves the cross-SDK compatibility convention historically
// implemented inside the SDK's PersistentAuth.dualWrite.
//
// Mirroring happens inside Put, so callers do not need a separate write for
// the legacy host-based key.
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

// Put implements [Store]. Writes under the primary key are also mirrored under
// the host key (when distinct); writes under any other key pass through
// unchanged so that an explicit host-key write does not recursively re-expand.
//
// The host-key mirror is best-effort: if the second Put fails, the error
// is silently dropped. The host-key entry is a backward-compat shim for old
// Go SDK versions (v0.61-v0.103) that still look up by host. Failing the
// whole Put call would break primary login over a non-essential mirror,
// so a stale host-key entry is the lesser harm.
func (s *DualWritingStore) Put(key string, e Entry) error {
	if err := s.inner.Put(key, e); err != nil {
		return err
	}
	primaryKey := s.arg.GetCacheKey()
	if key != primaryKey {
		return nil
	}
	hostKey := hostCacheKey(s.arg)
	if hostKey == "" || hostKey == primaryKey {
		return nil
	}
	_ = s.inner.Put(hostKey, e)
	return nil
}

// Lookup implements [Store]; delegates to the inner store.
func (s *DualWritingStore) Lookup(key string) (Entry, error) {
	return s.inner.Lookup(key)
}

// Delete implements [Store]. Deletes under the primary key are also mirrored
// under the host key, using the same best-effort compatibility policy as Put.
func (s *DualWritingStore) Delete(key string) error {
	if err := s.inner.Delete(key); err != nil {
		return err
	}
	primaryKey := s.arg.GetCacheKey()
	if key != primaryKey {
		return nil
	}
	hostKey := hostCacheKey(s.arg)
	if hostKey == "" || hostKey == primaryKey {
		return nil
	}
	_ = s.inner.Delete(hostKey)
	return nil
}

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
