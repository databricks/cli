package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderReturnsIncreasingIndex(t *testing.T) {
	o := newOrder([]string{})
	assert.Equal(t, 1, o.get("a"))
	assert.Equal(t, 2, o.get("b"))
	assert.Equal(t, 3, o.get("c"))
}

func TestOrderReturnsNegativeIndexForPredefinedKeys(t *testing.T) {
	o := newOrder([]string{"a", "b", "c"})
	assert.Equal(t, -3, o.get("a"))
	assert.Equal(t, -2, o.get("b"))
	assert.Equal(t, -1, o.get("c"))
	assert.Equal(t, 1, o.get("d"))
	assert.Equal(t, 2, o.get("e"))
	assert.Equal(t, 3, o.get("f"))
}
