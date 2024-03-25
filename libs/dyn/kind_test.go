package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestKindZeroValue(t *testing.T) {
	// Assert that the zero value of [dyn.Kind] is the invalid kind.
	var k dyn.Kind
	assert.Equal(t, dyn.KindInvalid, k)
}

func TestKindToString(t *testing.T) {
	for _, tt := range []struct {
		k dyn.Kind
		s string
	}{
		{dyn.KindInvalid, "invalid"},
		{dyn.KindMap, "map"},
		{dyn.KindSequence, "sequence"},
		{dyn.KindString, "string"},
		{dyn.KindBool, "bool"},
		{dyn.KindInt, "int"},
		{dyn.KindFloat, "float"},
		{dyn.KindTime, "time"},
		{dyn.KindNil, "nil"},
	} {
		assert.Equal(t, tt.s, tt.k.String())
	}

	// Panic on unknown kind.
	assert.PanicsWithValue(t, "invalid kind value: 100", func() {
		_ = dyn.Kind(100).String()
	})
}
