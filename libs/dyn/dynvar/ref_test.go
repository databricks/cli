package dynvar

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/require"
)

func TestNewRefNoString(t *testing.T) {
	_, ok := NewRef(dyn.V(1))
	require.False(t, ok, "should not match non-string")
}
