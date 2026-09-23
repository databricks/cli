package structcopy

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type copyKindA string

type copyKindB string

type copySource struct {
	Name            string    `json:"name"`
	Extra           string    `json:"extra"` // absent from dst -> dropped
	Kind            copyKindA `json:"kind"`  // same underlying type as dst, distinct name -> converted
	ForceSendFields []string
}

type copyDest struct {
	Name            string    `json:"name"`
	Missing         string    `json:"missing"` // absent from src -> left zero
	Kind            copyKindB `json:"kind"`
	ForceSendFields []string
}

func TestCopy(t *testing.T) {
	copier, err := Compile(reflect.TypeFor[*copySource](), reflect.TypeFor[*copyDest]())
	require.NoError(t, err)

	src := &copySource{
		Name:            "n",
		Extra:           "e",
		Kind:            "classic",
		ForceSendFields: []string{"Name", "Extra", "Kind"},
	}
	got := copier.Copy(src).(*copyDest)

	assert.Equal(t, "n", got.Name)
	assert.Empty(t, got.Missing)                                   // absent from source
	assert.Equal(t, copyKindB("classic"), got.Kind)                // converted across enum types
	assert.Equal(t, []string{"Name", "Kind"}, got.ForceSendFields) // "Extra" filtered out (not a dst field)
}

type unsafeSource struct {
	N int `json:"n"`
}

type unsafeDest struct {
	N string `json:"n"`
}

func TestCompileRejectsUnsafe(t *testing.T) {
	_, err := Compile(reflect.TypeFor[*unsafeSource](), reflect.TypeFor[*unsafeDest]())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not assignable or safely convertible")
}

func TestSafeConvert(t *testing.T) {
	tests := []struct {
		name string
		dst  reflect.Type
		src  reflect.Type
		want bool
	}{
		{
			name: "identical",
			dst:  reflect.TypeFor[string](),
			src:  reflect.TypeFor[string](),
			want: true,
		},
		{
			name: "same underlying string enums",
			dst:  reflect.TypeFor[copyKindA](),
			src:  reflect.TypeFor[copyKindB](),
			want: true,
		},
		{
			name: "int to string",
			dst:  reflect.TypeFor[string](),
			src:  reflect.TypeFor[int](),
			want: false,
		},
		{
			name: "int64 to int32",
			dst:  reflect.TypeFor[int32](),
			src:  reflect.TypeFor[int64](),
			want: false,
		},
		{
			name: "bytes to string",
			dst:  reflect.TypeFor[string](),
			src:  reflect.TypeFor[[]byte](),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, safeConvert(tt.dst, tt.src))
		})
	}
}
