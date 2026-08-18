package dms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceKeyDropsAndRestoresTheStatePrefix(t *testing.T) {
	key := KeyFromState("resources.jobs.foo")
	assert.Equal(t, ResourceKey("jobs.foo"), key)
	assert.Equal(t, "resources.jobs.foo", key.StateKey())
}

func TestResourceKeyFromAKeyWithoutThePrefixIsUnchanged(t *testing.T) {
	// The service never sends the prefix, so a key read back converts as-is.
	assert.Equal(t, ResourceKey("jobs.foo"), KeyFromState("jobs.foo"))
}
