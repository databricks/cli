package convert

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromTypedStructZeroFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{}
	ref := dyn.NilValue

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{}), nv)
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

func TestFromTypedStructSetFieldsRetainLocationIfUnchanged(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{
		Foo: "bar",
		Bar: "qux",
	}

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.NewValue("bar", dyn.Location{File: "foo"}),
		"bar": dyn.NewValue("baz", dyn.Location{File: "bar"}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo has retained its location.
	assert.Equal(t, dyn.NewValue("bar", dyn.Location{File: "foo"}), nv.Get("foo"))

	// Assert bar lost its location (because it was overwritten).
	assert.Equal(t, dyn.NewValue("qux", dyn.Location{}), nv.Get("bar"))
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
	var src = map[string]string{}

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.V("bar"),
		"bar": dyn.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{}), nv)
}

func TestFromTypedMapNonEmpty(t *testing.T) {
	var src = map[string]string{
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

func TestFromTypedMapNonEmptyRetainLocationIfUnchanged(t *testing.T) {
	var src = map[string]string{
		"foo": "bar",
		"bar": "qux",
	}

	ref := dyn.V(map[string]dyn.Value{
		"foo": dyn.NewValue("bar", dyn.Location{File: "foo"}),
		"bar": dyn.NewValue("baz", dyn.Location{File: "bar"}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo has retained its location.
	assert.Equal(t, dyn.NewValue("bar", dyn.Location{File: "foo"}), nv.Get("foo"))

	// Assert bar lost its location (because it was overwritten).
	assert.Equal(t, dyn.NewValue("qux", dyn.Location{}), nv.Get("bar"))
}

func TestFromTypedMapFieldWithZeroValue(t *testing.T) {
	var src = map[string]string{
		"foo": "",
	}

	ref := dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(map[string]dyn.Value{
		"foo": dyn.NilValue,
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
	var src = []string{}

	ref := dyn.V([]dyn.Value{
		dyn.V("bar"),
		dyn.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V([]dyn.Value{}), nv)
}

func TestFromTypedSliceNonEmpty(t *testing.T) {
	var src = []string{
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

func TestFromTypedSliceNonEmptyRetainLocationIfUnchanged(t *testing.T) {
	var src = []string{
		"foo",
		"bar",
	}

	ref := dyn.V([]dyn.Value{
		dyn.NewValue("foo", dyn.Location{File: "foo"}),
		dyn.NewValue("baz", dyn.Location{File: "baz"}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo has retained its location.
	assert.Equal(t, dyn.NewValue("foo", dyn.Location{File: "foo"}), nv.Index(0))

	// Assert bar lost its location (because it was overwritten).
	assert.Equal(t, dyn.NewValue("bar", dyn.Location{}), nv.Index(1))
}

func TestFromTypedStringEmpty(t *testing.T) {
	var src string
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedStringEmptyOverwrite(t *testing.T) {
	var src string
	var ref = dyn.V("old")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(""), nv)
}

func TestFromTypedStringNonEmpty(t *testing.T) {
	var src string = "new"
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("new"), nv)
}

func TestFromTypedStringNonEmptyOverwrite(t *testing.T) {
	var src string = "new"
	var ref = dyn.V("old")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V("new"), nv)
}

func TestFromTypedStringRetainsLocationsIfUnchanged(t *testing.T) {
	var src string = "foo"
	var ref = dyn.NewValue("foo", dyn.Location{File: "foo"})
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue("foo", dyn.Location{File: "foo"}), nv)
}

func TestFromTypedStringTypeError(t *testing.T) {
	var src string = "foo"
	var ref = dyn.V(1234)
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedBoolEmpty(t *testing.T) {
	var src bool
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedBoolEmptyOverwrite(t *testing.T) {
	var src bool
	var ref = dyn.V(true)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(false), nv)
}

func TestFromTypedBoolNonEmpty(t *testing.T) {
	var src bool = true
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(true), nv)
}

func TestFromTypedBoolNonEmptyOverwrite(t *testing.T) {
	var src bool = true
	var ref = dyn.V(false)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(true), nv)
}

func TestFromTypedBoolRetainsLocationsIfUnchanged(t *testing.T) {
	var src bool = true
	var ref = dyn.NewValue(true, dyn.Location{File: "foo"})
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(true, dyn.Location{File: "foo"}), nv)
}

func TestFromTypedBoolTypeError(t *testing.T) {
	var src bool = true
	var ref = dyn.V("string")
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedIntEmpty(t *testing.T) {
	var src int
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedIntEmptyOverwrite(t *testing.T) {
	var src int
	var ref = dyn.V(1234)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(int64(0)), nv)
}

func TestFromTypedIntNonEmpty(t *testing.T) {
	var src int = 1234
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(int64(1234)), nv)
}

func TestFromTypedIntNonEmptyOverwrite(t *testing.T) {
	var src int = 1234
	var ref = dyn.V(1233)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(int64(1234)), nv)
}

func TestFromTypedIntRetainsLocationsIfUnchanged(t *testing.T) {
	var src int = 1234
	var ref = dyn.NewValue(1234, dyn.Location{File: "foo"})
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(1234, dyn.Location{File: "foo"}), nv)
}

func TestFromTypedIntTypeError(t *testing.T) {
	var src int = 1234
	var ref = dyn.V("string")
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}

func TestFromTypedFloatEmpty(t *testing.T) {
	var src float64
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NilValue, nv)
}

func TestFromTypedFloatEmptyOverwrite(t *testing.T) {
	var src float64
	var ref = dyn.V(1.23)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(0.0), nv)
}

func TestFromTypedFloatNonEmpty(t *testing.T) {
	var src float64 = 1.23
	var ref = dyn.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(1.23), nv)
}

func TestFromTypedFloatNonEmptyOverwrite(t *testing.T) {
	var src float64 = 1.23
	var ref = dyn.V(1.24)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.V(1.23), nv)
}

func TestFromTypedFloatRetainsLocationsIfUnchanged(t *testing.T) {
	var src float64 = 1.23
	var ref = dyn.NewValue(1.23, dyn.Location{File: "foo"})
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, dyn.NewValue(1.23, dyn.Location{File: "foo"}), nv)
}

func TestFromTypedFloatTypeError(t *testing.T) {
	var src float64 = 1.23
	var ref = dyn.V("string")
	_, err := FromTyped(src, ref)
	require.Error(t, err)
}
