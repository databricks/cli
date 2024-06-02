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

	t.Run("pointer to a empty string", func(t *testing.T) {
		s := ""
		assertFromTypedToTypedEqual(t, &s)
	})

	t.Run("nil pointer", func(t *testing.T) {
		var s *string
		assertFromTypedToTypedEqual(t, s)
	})

	t.Run("pointer to struct with scalar values", func(t *testing.T) {
		s := ""
		type foo struct {
			A string  `json:"a"`
			B int     `json:"b"`
			C bool    `json:"c"`
			D *string `json:"d"`
		}
		assertFromTypedToTypedEqual(t, &foo{
			A: "a",
			B: 1,
			C: true,
			D: &s,
		})
		assertFromTypedToTypedEqual(t, &foo{
			A: "",
			B: 0,
			C: false,
			D: nil,
		})
	})

	t.Run("map with scalar values", func(t *testing.T) {
		assertFromTypedToTypedEqual(t, map[string]string{
			"a": "a",
			"b": "b",
			"c": "",
		})
		assertFromTypedToTypedEqual(t, map[string]int{
			"a": 1,
			"b": 0,
			"c": 2,
		})
	})
}
