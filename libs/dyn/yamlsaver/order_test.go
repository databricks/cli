package yamlsaver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrderReturnsIncreasingIndex(t *testing.T) {
	o := NewOrder([]string{})
	assert.Equal(t, 1, o.Get("a"))
	assert.Equal(t, 2, o.Get("b"))
	assert.Equal(t, 3, o.Get("c"))
}

func TestOrderReturnsNegativeIndexForPredefinedKeys(t *testing.T) {
	o := NewOrder([]string{"a", "b", "c"})
	assert.Equal(t, -3, o.Get("a"))
	assert.Equal(t, -2, o.Get("b"))
	assert.Equal(t, -1, o.Get("c"))
	assert.Equal(t, 1, o.Get("d"))
	assert.Equal(t, 2, o.Get("e"))
	assert.Equal(t, 3, o.Get("f"))
}
