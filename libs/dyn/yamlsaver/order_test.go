package yamlsaver

import (
	"testing"

	"github.com/databricks/cli/libs/dyn/dynassert"
)

func TestOrderReturnsIncreasingIndex(t *testing.T) {
	o := NewOrder([]string{})
	dynassert.Equal(t, 1, o.Get("a"))
	dynassert.Equal(t, 2, o.Get("b"))
	dynassert.Equal(t, 3, o.Get("c"))
}

func TestOrderReturnsNegativeIndexForPredefinedKeys(t *testing.T) {
	o := NewOrder([]string{"a", "b", "c"})
	dynassert.Equal(t, -3, o.Get("a"))
	dynassert.Equal(t, -2, o.Get("b"))
	dynassert.Equal(t, -1, o.Get("c"))
	dynassert.Equal(t, 1, o.Get("d"))
	dynassert.Equal(t, 2, o.Get("e"))
	dynassert.Equal(t, 3, o.Get("f"))
}
