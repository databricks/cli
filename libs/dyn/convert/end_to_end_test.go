package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func assertFromTypedToTypedEqual[T any](t *testing.T, src T) {
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	var dst T
	err = ToTyped(&dst, nv)
	require.NoError(t, err)
	assert.Equal(t, src, dst)
}

func TestAdditional(t *testing.T) {
	type StructType struct {
		Str string `json:"str"`
	}

	type Tmp struct {
		MapToPointer   map[string]*string `json:"map_to_pointer"`
		SliceOfPointer []*string          `json:"slice_of_pointer"`
		NestedStruct   StructType         `json:"nested_struct"`
	}

	t.Run("nil", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{})
	})

	t.Run("empty map", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			MapToPointer: map[string]*string{},
		})
	})

	t.Run("map with empty string value", func(t *testing.T) {
		s := ""
		assertFromTypedToTypedEqual(t, Tmp{
			MapToPointer: map[string]*string{
				"key": &s,
			},
		})
	})

	t.Run("map with nil value", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			MapToPointer: map[string]*string{
				"key": nil,
			},
		})
	})

	t.Run("empty slice", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			SliceOfPointer: []*string{},
		})
	})

	t.Run("slice with nil value", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, Tmp{
			SliceOfPointer: []*string{nil},
		})
	})
}
