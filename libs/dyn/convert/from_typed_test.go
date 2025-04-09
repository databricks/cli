package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
)

func TestFromTypedStructZeroFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{}

	// For an empty struct with a nil reference we expect a nil.
	nv, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)

	// For an empty struct with a non-nil reference we expect an empty map.
	nv, err = FromTyped(src, dyn.V(map[string]dyn.Value{}))
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{}), nv)
}

func TestFromTypedStructPointerZeroFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	var src *Tmp
	var nv dyn.Value
	var err error

	// For a nil pointer with a nil reference we expect a nil.
	src = nil
	nv, err = FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)

	// For a nil pointer with a non-nil reference we expect a nil.
	src = nil
	nv, err = FromTyped(src, dyn.V(map[string]dyn.Value{}))
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)

	// For an initialized pointer with a nil reference we expect an empty map.
	src = &Tmp{}
	nv, err = FromTyped(src, dyn.NilValue)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{}), nv)

	// For an initialized pointer with a non-nil reference we expect an empty map.
	src = &Tmp{}
	nv, err = FromTyped(src, dyn.V(map[string]dyn.Value{}))
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{}), nv)
}

func TestFromTypedStructNilFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	// For a zero value struct with a reference containing nil fields we expect the nils to be retained.
	src := Tmp{}
	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.NilValue,
		"bar": dyn.NilValue,
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.NilValue,
		"bar": dyn.NilValue,
	}), nv)
}

func TestFromTypedStructSetFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{
		Foo: "foo",
		Bar: "bar",
	}

	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V("foo"),
		"bar": dyn.V("bar"),
	}), nv)
}

func TestFromTypedStructSetFieldsRetainLocation(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{
		Foo: "bar",
		Bar: "qux",
	}

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.NewValue("bar", []dyn.Location{{File: "foo"}}),
		"bar": dyn.NewValue("baz", []dyn.Location{{File: "bar"}}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo and bar have retained their location.
	assert.Equal(t, dyn.NewValue("bar", []dyn.Location{{File: "foo"}}), nv.Get("foo"))
	assert.Equal(t, dyn.NewValue("qux", []dyn.Location{{File: "bar"}}), nv.Get("bar"))
}

func TestFromTypedStringMapWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := map[string]string{
		"foo": "",
		"bar": "fuzz",
	}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(""),
		"bar": dyn.V("fuzz"),
	}), nv)
}

func TestFromTypedStringSliceWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := []string{"a", "", "c"}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{
		dyn.V("a"), dyn.V(""), dyn.V("c"),
	}), nv)
}

func TestFromTypedStringStructWithZeroValue(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	ref := dyn.NilValue
	src := Tmp{
		Foo: "foo",
		Bar: "",
	}

	// Note, the zero value is not included in the output.
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V("foo"),
	}), nv)
}

func TestFromTypedBoolMapWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := map[string]bool{
		"foo": false,
		"bar": true,
	}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(false),
		"bar": dyn.V(true),
	}), nv)
}

func TestFromTypedBoolSliceWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := []bool{true, false, true}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{
		dyn.V(true), dyn.V(false), dyn.V(true),
	}), nv)
}

func TestFromTypedBoolStructWithZeroValue(t *testing.T) {
	type Tmp struct {
		Foo bool `json:"foo"`
		Bar bool `json:"bar"`
	}

	ref := dyn.NilValue
	src := Tmp{
		Foo: true,
		Bar: false,
	}

	// Note, the zero value is not included in the output.
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(true),
	}), nv)
}

func TestFromTypedIntMapWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := map[string]int{
		"foo": 0,
		"bar": 1,
	}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(int64(0)),
		"bar": dyn.V(int64(1)),
	}), nv)
}

func TestFromTypedIntSliceWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := []int{1, 0, 2}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{
		dyn.V(int64(1)), dyn.V(int64(0)), dyn.V(int64(2)),
	}), nv)
}

func TestFromTypedIntStructWithZeroValue(t *testing.T) {
	type Tmp struct {
		Foo int `json:"foo"`
		Bar int `json:"bar"`
	}

	ref := dyn.NilValue
	src := Tmp{
		Foo: 1,
		Bar: 0,
	}

	// Note, the zero value is not included in the output.
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(int64(1)),
	}), nv)
}

func TestFromTypedFloatMapWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := map[string]float64{
		"foo": 0.0,
		"bar": 1.0,
	}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(0.0),
		"bar": dyn.V(1.0),
	}), nv)
}

func TestFromTypedFloatSliceWithZeroValue(t *testing.T) {
	ref := dyn.NilValue
	src := []float64{1.0, 0.0, 2.0}

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{
		dyn.V(1.0), dyn.V(0.0), dyn.V(2.0),
	}), nv)
}

func TestFromTypedFloatStructWithZeroValue(t *testing.T) {
	type Tmp struct {
		Foo float64 `json:"foo"`
		Bar float64 `json:"bar"`
	}

	ref := dyn.NilValue
	src := Tmp{
		Foo: 1.0,
		Bar: 0.0,
	}

	// Note, the zero value is not included in the output.
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(1.0),
	}), nv)
}

func TestFromTypedMapNil(t *testing.T) {
	var src map[string]string = nil

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedMapEmpty(t *testing.T) {
	src := map[string]string{}

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{}), nv)
}

func TestFromTypedMapNonEmpty(t *testing.T) {
	src := map[string]string{
		"foo": "foo",
		"bar": "bar",
	}

	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V("foo"),
		"bar": dyn.V("bar"),
	}), nv)
}

func TestFromTypedMapNonEmptyRetainLocation(t *testing.T) {
	src := map[string]string{
		"foo": "bar",
		"bar": "qux",
	}

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.NewValue("bar", []dyn.Location{{File: "foo"}}),
		"bar": dyn.NewValue("baz", []dyn.Location{{File: "bar"}}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo and bar have retained their locations.
	assert.Equal(t, dyn.NewValue("bar", []dyn.Location{{File: "foo"}}), nv.Get("foo"))
	assert.Equal(t, dyn.NewValue("qux", []dyn.Location{{File: "bar"}}), nv.Get("bar"))
}

func TestFromTypedMapFieldWithZeroValue(t *testing.T) {
	src := map[string]string{
		"foo": "",
	}

	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V(""),
	}), nv)
}

func TestFromTypedSliceNil(t *testing.T) {
	var src []string = nil

	ref := dyn.V([]dyn.Value{
		dyn.V("bar"),
		dyn.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedSliceEmpty(t *testing.T) {
	//nolint:gocritic
	src := []string{}

	ref := dyn.V([]dyn.Value{
		dyn.V("bar"),
		dyn.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{}), nv)
}

func TestFromTypedSliceNonEmpty(t *testing.T) {
	src := []string{
		"foo",
		"bar",
	}

	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{
		dyn.V("foo"),
		dyn.V("bar"),
	}), nv)
}

func TestFromTypedSliceNonEmptyRetainLocation(t *testing.T) {
	src := []string{
		"foo",
		"bar",
	}

	ref := dyn.V([]dyn.Value{
		dyn.NewValue("foo", []dyn.Location{{File: "foo"}}),
		dyn.NewValue("bar", []dyn.Location{{File: "bar"}}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo and bar have retained their locations.
	assert.Equal(t, dyn.NewValue("foo", []dyn.Location{{File: "foo"}}), nv.Index(0))
	assert.Equal(t, dyn.NewValue("bar", []dyn.Location{{File: "bar"}}), nv.Index(1))
}

func TestFromTypedStringEmpty(t *testing.T) {
	var src string
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedStringEmptyOverwrite(t *testing.T) {
	var src string
	ref := dyn.V("old")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(""), nv)
}

func TestFromTypedStringNonEmpty(t *testing.T) {
	var src string = "new"
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("new"), nv)
}

func TestFromTypedStringNonEmptyOverwrite(t *testing.T) {
	var src string = "new"
	ref := dyn.V("old")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("new"), nv)
}

func TestFromTypedStringRetainsLocations(t *testing.T) {
	ref := dyn.NewValue("foo", []dyn.Location{{File: "foo"}})

	// case: value has not been changed
	var src string = "foo"
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue("foo", []dyn.Location{{File: "foo"}}), nv)

	// case: value has been changed
	src = "bar"
	nv, err = FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue("bar", []dyn.Location{{File: "foo"}}), nv)
}

func TestFromTypedStringTypeError(t *testing.T) {
	var src string = "foo"
	ref := dyn.V(1234)
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedBoolEmpty(t *testing.T) {
	var src bool
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedBoolEmptyOverwrite(t *testing.T) {
	var src bool
	ref := dyn.V(true)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(false), nv)
}

func TestFromTypedBoolNonEmpty(t *testing.T) {
	var src bool = true
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(true), nv)
}

func TestFromTypedBoolNonEmptyOverwrite(t *testing.T) {
	var src bool = true
	ref := dyn.V(false)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(true), nv)
}

func TestFromTypedBoolRetainsLocations(t *testing.T) {
	ref := dyn.NewValue(true, []dyn.Location{{File: "foo"}})

	// case: value has not been changed
	var src bool = true
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(true, []dyn.Location{{File: "foo"}}), nv)

	// case: value has been changed
	src = false
	nv, err = FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(false, []dyn.Location{{File: "foo"}}), nv)
}

func TestFromTypedBoolVariableReference(t *testing.T) {
	var src bool = true
	ref := dyn.V("${var.foo}")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("${var.foo}"), nv)
}

func TestFromTypedBoolTypeError(t *testing.T) {
	var src bool = true
	ref := dyn.V("string")
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedIntEmpty(t *testing.T) {
	var src int
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedIntEmptyOverwrite(t *testing.T) {
	var src int
	ref := dyn.V(1234)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(int64(0)), nv)
}

func TestFromTypedIntNonEmpty(t *testing.T) {
	var src int = 1234
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(int64(1234)), nv)
}

func TestFromTypedIntNonEmptyOverwrite(t *testing.T) {
	var src int = 1234
	ref := dyn.V(1233)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(int64(1234)), nv)
}

func TestFromTypedIntRetainsLocations(t *testing.T) {
	ref := dyn.NewValue(1234, []dyn.Location{{File: "foo"}})

	// case: value has not been changed
	var src int = 1234
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(1234, []dyn.Location{{File: "foo"}}), nv)

	// case: value has been changed
	src = 1235
	nv, err = FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(int64(1235), []dyn.Location{{File: "foo"}}), nv)
}

func TestFromTypedIntVariableReference(t *testing.T) {
	var src int = 1234
	ref := dyn.V("${var.foo}")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("${var.foo}"), nv)
}

func TestFromTypedIntTypeError(t *testing.T) {
	var src int = 1234
	ref := dyn.V("string")
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedFloatEmpty(t *testing.T) {
	var src float64
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedFloatEmptyOverwrite(t *testing.T) {
	var src float64
	ref := dyn.V(1.23)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(0.0), nv)
}

func TestFromTypedFloatNonEmpty(t *testing.T) {
	var src float64 = 1.23
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(1.23), nv)
}

func TestFromTypedFloatNonEmptyOverwrite(t *testing.T) {
	var src float64 = 1.23
	ref := dyn.V(1.24)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(1.23), nv)
}

func TestFromTypedFloatRetainsLocations(t *testing.T) {
	var src float64
	ref := dyn.NewValue(1.23, []dyn.Location{{File: "foo"}})

	// case: value has not been changed
	src = 1.23
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(1.23, []dyn.Location{{File: "foo"}}), nv)

	// case: value has been changed
	src = 1.24
	nv, err = FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(1.24, []dyn.Location{{File: "foo"}}), nv)
}

func TestFromTypedFloatVariableReference(t *testing.T) {
	var src float64 = 1.23
	ref := dyn.V("${var.foo}")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("${var.foo}"), nv)
}

func TestFromTypedFloatTypeError(t *testing.T) {
	var src float64 = 1.23
	ref := dyn.V("string")
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedAny(t *testing.T) {
	type Tmp struct {
		Foo any `json:"foo"`
		Bar any `json:"bar"`
		Foz any `json:"foz"`
		Baz any `json:"baz"`
	}

	src := Tmp{
		Foo: "foo",
		Bar: false,
		Foz: 0,
		Baz: map[string]any{
			"foo": "foo",
			"bar": 1234,
			"qux": 0,
			"nil": nil,
		},
	}

	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.V("foo"),
		"bar": dyn.V(false),
		"foz": dyn.V(int64(0)),
		"baz": dyn.V(map[string]dyn.Value{
			"foo": dyn.V("foo"),
			"bar": dyn.V(int64(1234)),
			"qux": dyn.V(int64(0)),
			"nil": dyn.V(nil),
		}),
	}), nv)
}

func TestFromTypedAnyNil(t *testing.T) {
	var src any = nil
	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedNilPointerRetainsLocations(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	var src *Tmp
	ref := dyn.NewValue(nil, []dyn.Location{{File: "foobar"}})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(nil, []dyn.Location{{File: "foobar"}}), nv)
}

func TestFromTypedNilMapRetainsLocation(t *testing.T) {
	var src map[string]string
	ref := dyn.NewValue(nil, []dyn.Location{{File: "foobar"}})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(nil, []dyn.Location{{File: "foobar"}}), nv)
}

func TestFromTypedNilSliceRetainsLocation(t *testing.T) {
	var src []string
	ref := dyn.NewValue(nil, []dyn.Location{{File: "foobar"}})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(nil, []dyn.Location{{File: "foobar"}}), nv)
}
