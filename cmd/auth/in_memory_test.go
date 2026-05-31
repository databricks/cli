package auth

import (
	"github.com/databricks/cli/libs/auth/storage"
	"golang.org/x/oauth2"
)

type inMemoryTokenCache struct {
	Tokens map[string]*oauth2.Token
}

// Lookup returns a copy to match real (file-backed) cache behavior, where
// each lookup deserializes a fresh struct. Without the copy, callers that
// mutate the returned token (e.g. clearing RefreshToken) would corrupt
// entries shared across test cases.
func (i *inMemoryTokenCache) Lookup(key string) (storage.Entry, error) {
	token, ok := i.Tokens[key]
	if !ok {
		return storage.Entry{}, storage.ErrNotFound
	}
	cp := *token
	return storage.Entry{Token: &cp}, nil
}

// Put stores a copy to prevent callers from mutating cached entries after
// put returns (mirrors file-backed cache semantics).
func (i *inMemoryTokenCache) Put(key string, e storage.Entry) error {
	cp := *e.Token
	i.Tokens[key] = &cp
	return nil
}

// Delete deletes the entry under key. Deleting a missing entry is not
// an error.
func (i *inMemoryTokenCache) Delete(key string) error {
	delete(i.Tokens, key)
	return nil
}

var _ storage.Store = (*inMemoryTokenCache)(nil)
