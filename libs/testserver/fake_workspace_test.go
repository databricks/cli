package testserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Stores are keyed by type, so a second registration of the same type would
// silently replace the first.
func TestStoresAreRegisteredOncePerType(t *testing.T) {
	workspace := NewFakeWorkspace("http://test", "dbapi123")
	assert.Len(t, workspace.stores, len(storeRegistrations))
}
