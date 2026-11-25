package testserver

import (
	"fmt"
	"testing"
)

// Stores are keyed by type, so a second registration of the same type would
// silently replace the first.
func TestStoresAreRegisteredOncePerType(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	keysByType := map[string][]string{}
	for _, r := range storeRegistrations {
		_, store := r.newStore(workspace)
		storeType := fmt.Sprintf("%T", store)
		keysByType[storeType] = append(keysByType[storeType], r.key)
	}
	for storeType, keys := range keysByType {
		if len(keys) > 1 {
			t.Errorf("store type %s is registered %d times, with keys %v", storeType, len(keys), keys)
		}
	}
}
