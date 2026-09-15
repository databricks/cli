package storage

import (
	"sync"

	"golang.org/x/oauth2"
)

// NewMemoryStore returns a Store that stores entries in process
// memory only. Tokens do not persist across process restarts. This is the
// default store used by PersistentAuth when no WithTokenStore option is
// provided. Most production consumers should supply a persistent Store.
func NewMemoryStore() Store {
	return &memoryStore{entries: map[string]Entry{}}
}

type memoryStore struct {
	mu      sync.Mutex
	entries map[string]Entry
}

func cloneToken(t *oauth2.Token) *oauth2.Token {
	if t == nil {
		return nil
	}
	clone := *t
	return &clone
}

func cloneEntry(e Entry) Entry {
	e.Token = cloneToken(e.Token)
	return e
}

// Put implements Store.
func (s *memoryStore) Put(key string, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = cloneEntry(e)
	return nil
}

// Lookup implements Store. Returns ErrNotFound when the key is absent.
func (s *memoryStore) Lookup(key string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return cloneEntry(e), nil
}

// Delete implements Store.
func (s *memoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}
