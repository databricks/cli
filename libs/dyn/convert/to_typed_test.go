package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestToTypedStruct(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar,omitempty"`

		// Baz must be skipped.
		Baz string `json:""`

		// Qux must be skipped.
		Qux string `json:"-"`
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, "bar", out.Foo)
	assert.Equal(t, "baz", out.Bar)
}

func TestToTypedStructOverwrite(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar,omitempty"`

		// Baz must be skipped.
		Baz string `json:""`

		// Qux must be skipped.
		Qux string `json:"-"`
	}

	out := Tmp{
		Foo: "baz",
		Bar: "qux",
	}
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, "bar", out.Foo)
	assert.Equal(t, "baz", out.Bar)
}

func TestToTypedStructClearFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar,omitempty"`
	}

	// Struct value with non-empty fields.
	out := Tmp{
		Foo: "baz",
		Bar: "qux",
	}

	// Value is an empty map.
	v := dyn.V(map[string]dyn.Value{})

	// The previously set fields should be cleared.
	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, Tmp{}, out)
}

func TestToTypedStructAnonymousByValue(t *testing.T) {
	type Bar struct {
		Bar string `json:"bar"`
	}

	type Foo struct {
		Foo string `json:"foo"`
		Bar
	}

	type Tmp struct {
		Foo
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, "bar", out.Foo.Foo)
	assert.Equal(t, "baz", out.Foo.Bar.Bar)
}

func TestToTypedStructAnonymousByPointer(t *testing.T) {
	type Bar struct {
		Bar string `json:"bar"`
	}

	type Foo struct {
		Foo string `json:"foo"`
		*Bar
	}

	type Tmp struct {
		*Foo
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, "bar", out.Foo.Foo)
	assert.Equal(t, "baz", out.Foo.Bar.Bar)
}

func TestToTypedStructNil(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	out := Tmp{}
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, Tmp{}, out)
}

func TestToTypedStructNilOverwrite(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	out := Tmp{"bar"}
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, Tmp{}, out)
}

func TestToTypedStructWithValueField(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`

		ConfigValue dyn.Value
	}

	var out Tmp
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, "bar", out.Foo)
	assert.Equal(t, v, out.ConfigValue)
}

func TestToTypedMap(t *testing.T) {
	out := map[string]string{}

	v := dyn.V(map[string]dyn.Value{
		"key": dyn.V("value"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "value", out["key"])
}

func TestToTypedMapOverwrite(t *testing.T) {
	out := map[string]string{
		"foo": "bar",
	}

	v := dyn.V(map[string]dyn.Value{
		"bar": dyn.V("qux"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "qux", out["bar"])
}

func TestToTypedMapWithPointerElement(t *testing.T) {
	var out map[string]*string

	v := dyn.V(map[string]dyn.Value{
		"key": dyn.V("value"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "value", *out["key"])
}

func TestToTypedMapNil(t *testing.T) {
	out := map[string]string{}
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedMapNilOverwrite(t *testing.T) {
	out := map[string]string{
		"foo": "bar",
	}
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedSlice(t *testing.T) {
	var out []string

	v := dyn.V([]dyn.Value{
		dyn.V("foo"),
		dyn.V("bar"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 2)
	assert.Equal(t, "foo", out[0])
	assert.Equal(t, "bar", out[1])
}

func TestToTypedSliceOverwrite(t *testing.T) {
	out := []string{"qux"}

	v := dyn.V([]dyn.Value{
		dyn.V("foo"),
		dyn.V("bar"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 2)
	assert.Equal(t, "foo", out[0])
	assert.Equal(t, "bar", out[1])
}

func TestToTypedSliceWithPointerElement(t *testing.T) {
	var out []*string

	v := dyn.V([]dyn.Value{
		dyn.V("foo"),
		dyn.V("bar"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 2)
	assert.Equal(t, "foo", *out[0])
	assert.Equal(t, "bar", *out[1])
}

func TestToTypedSliceNil(t *testing.T) {
	var out []string
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedSliceNilOverwrite(t *testing.T) {
	out := []string{"foo"}
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestToTypedString(t *testing.T) {
	var out string
	err := ToTyped(&out, dyn.V("foo"))
	require.NoError(t, err)
	assert.Equal(t, "foo", out)
}

func TestToTypedStringOverwrite(t *testing.T) {
	var out string = "bar"
	err := ToTyped(&out, dyn.V("foo"))
	require.NoError(t, err)
	assert.Equal(t, "foo", out)
}

func TestToTypedStringFromBool(t *testing.T) {
	var out string
	err := ToTyped(&out, dyn.V(true))
	require.NoError(t, err)
	assert.Equal(t, "true", out)
}

func TestToTypedStringFromInt(t *testing.T) {
	var out string
	err := ToTyped(&out, dyn.V(123))
	require.NoError(t, err)
	assert.Equal(t, "123", out)
}

func TestToTypedStringFromFloat(t *testing.T) {
	var out string
	err := ToTyped(&out, dyn.V(1.2))
	require.NoError(t, err)
	assert.Equal(t, "1.2", out)
}

func TestToTypedBool(t *testing.T) {
	var out bool
	err := ToTyped(&out, dyn.V(true))
	require.NoError(t, err)
	assert.Equal(t, true, out)
}

func TestToTypedBoolOverwrite(t *testing.T) {
	var out bool = true
	err := ToTyped(&out, dyn.V(false))
	require.NoError(t, err)
	assert.Equal(t, false, out)
}

func TestToTypedBoolFromString(t *testing.T) {
	var out bool

	// True-ish
	for _, v := range []string{"y", "yes", "on", "true"} {
		err := ToTyped(&out, dyn.V(v))
		require.NoError(t, err)
		assert.Equal(t, true, out)
	}

	// False-ish
	for _, v := range []string{"n", "no", "off", "false"} {
		err := ToTyped(&out, dyn.V(v))
		require.NoError(t, err)
		assert.Equal(t, false, out)
	}

	// Other
	err := ToTyped(&out, dyn.V("some other string"))
	require.Error(t, err)
}

func TestToTypedBoolFromStringVariableReference(t *testing.T) {
	var out bool = true
	err := ToTyped(&out, dyn.V("${var.foo}"))
	require.NoError(t, err)
	assert.Equal(t, false, out)
}

func TestToTypedInt(t *testing.T) {
	var out int
	err := ToTyped(&out, dyn.V(1234))
	require.NoError(t, err)
	assert.Equal(t, int(1234), out)
}

func TestToTypedInt32(t *testing.T) {
	var out32 int32
	err := ToTyped(&out32, dyn.V(1235))
	require.NoError(t, err)
	assert.Equal(t, int32(1235), out32)
}

func TestToTypedInt64(t *testing.T) {
	var out64 int64
	err := ToTyped(&out64, dyn.V(1236))
	require.NoError(t, err)
	assert.Equal(t, int64(1236), out64)
}

func TestToTypedIntOverwrite(t *testing.T) {
	var out int = 123
	err := ToTyped(&out, dyn.V(1234))
	require.NoError(t, err)
	assert.Equal(t, int(1234), out)
}

func TestToTypedInt32Overwrite(t *testing.T) {
	var out32 int32 = 123
	err := ToTyped(&out32, dyn.V(1234))
	require.NoError(t, err)
	assert.Equal(t, int32(1234), out32)
}

func TestToTypedInt64Overwrite(t *testing.T) {
	var out64 int64 = 123
	err := ToTyped(&out64, dyn.V(1234))
	require.NoError(t, err)
	assert.Equal(t, int64(1234), out64)
}

func TestToTypedIntFromStringError(t *testing.T) {
	var out int
	err := ToTyped(&out, dyn.V("abc"))
	require.Error(t, err)
}

func TestToTypedIntFromStringInt(t *testing.T) {
	var out int
	err := ToTyped(&out, dyn.V("123"))
	require.NoError(t, err)
	assert.Equal(t, int(123), out)
}

func TestToTypedIntFromStringVariableReference(t *testing.T) {
	var out int = 123
	err := ToTyped(&out, dyn.V("${var.foo}"))
	require.NoError(t, err)
	assert.Equal(t, int(0), out)
}

func TestToTypedIntFromFloat(t *testing.T) {
	var out int
	err := ToTyped(&out, dyn.V(1.0))
	require.NoError(t, err)
	assert.Equal(t, int(1), out)
}

func TestToTypedIntFromFloatError(t *testing.T) {
	var out int
	err := ToTyped(&out, dyn.V(1.2))
	require.ErrorContains(t, err, "expected an int, found a float")
}

func TestToTypedFloat32(t *testing.T) {
	var out float32
	err := ToTyped(&out, dyn.V(float32(1.0)))
	require.NoError(t, err)
	assert.Equal(t, float32(1.0), out)
}

func TestToTypedFloat64(t *testing.T) {
	var out float64
	err := ToTyped(&out, dyn.V(float64(1.0)))
	require.NoError(t, err)
	assert.Equal(t, float64(1.0), out)
}

func TestToTypedFloat32Overwrite(t *testing.T) {
	var out float32 = 1.0
	err := ToTyped(&out, dyn.V(float32(2.0)))
	require.NoError(t, err)
	assert.Equal(t, float32(2.0), out)
}

func TestToTypedFloat64Overwrite(t *testing.T) {
	var out float64 = 1.0
	err := ToTyped(&out, dyn.V(float64(2.0)))
	require.NoError(t, err)
	assert.Equal(t, float64(2.0), out)
}

func TestToTypedFloat32FromStringError(t *testing.T) {
	var out float32
	err := ToTyped(&out, dyn.V("abc"))
	require.Error(t, err)
}

func TestToTypedFloat64FromStringError(t *testing.T) {
	var out float64
	err := ToTyped(&out, dyn.V("abc"))
	require.Error(t, err)
}

func TestToTypedFloat32FromString(t *testing.T) {
	var out float32
	err := ToTyped(&out, dyn.V("1.2"))
	require.NoError(t, err)
	assert.Equal(t, float32(1.2), out)
}

func TestToTypedFloat64FromString(t *testing.T) {
	var out float64
	err := ToTyped(&out, dyn.V("1.2"))
	require.NoError(t, err)
	assert.Equal(t, float64(1.2), out)
}

func TestToTypedFloat32FromStringVariableReference(t *testing.T) {
	var out float32 = 1.0
	err := ToTyped(&out, dyn.V("${var.foo}"))
	require.NoError(t, err)
	assert.Equal(t, float32(0.0), out)
}

func TestToTypedFloat64FromStringVariableReference(t *testing.T) {
	var out float64 = 1.0
	err := ToTyped(&out, dyn.V("${var.foo}"))
	require.NoError(t, err)
	assert.Equal(t, float64(0.0), out)
}

func TestToTypedWithAliasKeyType(t *testing.T) {
	type custom string

	var out map[custom]string
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Len(t, out, 2)
	assert.Equal(t, "bar", out["foo"])
	assert.Equal(t, "baz", out["bar"])
}

func TestToTypedAnyWithBool(t *testing.T) {
	var out any
	err := ToTyped(&out, dyn.V(false))
	require.NoError(t, err)
	assert.Equal(t, false, out)

	err = ToTyped(&out, dyn.V(true))
	require.NoError(t, err)
	assert.Equal(t, true, out)
}

func TestToTypedAnyWithMap(t *testing.T) {
	var out any
	v := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})
	err := ToTyped(&out, v)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"foo": "bar", "bar": "baz"}, out)
}

func TestToTypedAnyWithNil(t *testing.T) {
	var out any
	err := ToTyped(&out, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, nil, out)
}
