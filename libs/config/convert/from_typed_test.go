package convert

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromTypedStructZeroFields(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{}
	ref := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NilValue, nv)
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

	ref := config.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V(map[string]config.Value{
		"foo": config.V("foo"),
		"bar": config.V("bar"),
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

	ref := config.V(map[string]config.Value{
		"foo": config.NewValue("bar", config.Location{File: "foo"}),
		"bar": config.NewValue("baz", config.Location{File: "bar"}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo has retained its location.
	assert.Equal(t, config.NewValue("bar", config.Location{File: "foo"}), nv.Get("foo"))

	// Assert bar lost its location (because it was overwritten).
	assert.Equal(t, config.NewValue("qux", config.Location{}), nv.Get("bar"))
}

func TestFromTypedMapEmpty(t *testing.T) {
	var src = map[string]string{}

	ref := config.V(map[string]config.Value{
		"foo": config.V("bar"),
		"bar": config.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NilValue, nv)
}

func TestFromTypedMapNonEmpty(t *testing.T) {
	var src = map[string]string{
		"foo": "foo",
		"bar": "bar",
	}

	ref := config.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V(map[string]config.Value{
		"foo": config.V("foo"),
		"bar": config.V("bar"),
	}), nv)
}

func TestFromTypedMapNonEmptyRetainLocationIfUnchanged(t *testing.T) {
	var src = map[string]string{
		"foo": "bar",
		"bar": "qux",
	}

	ref := config.V(map[string]config.Value{
		"foo": config.NewValue("bar", config.Location{File: "foo"}),
		"bar": config.NewValue("baz", config.Location{File: "bar"}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo has retained its location.
	assert.Equal(t, config.NewValue("bar", config.Location{File: "foo"}), nv.Get("foo"))

	// Assert bar lost its location (because it was overwritten).
	assert.Equal(t, config.NewValue("qux", config.Location{}), nv.Get("bar"))
}

func TestFromTypedMapFieldWithZeroValue(t *testing.T) {
	var src = map[string]string{
		"foo": "",
	}

	ref := config.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V(map[string]config.Value{
		"foo": config.NilValue,
	}), nv)
}

func TestFromTypedSliceEmpty(t *testing.T) {
	var src = []string{}

	ref := config.V([]config.Value{
		config.V("bar"),
		config.V("baz"),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NilValue, nv)
}

func TestFromTypedSliceNonEmpty(t *testing.T) {
	var src = []string{
		"foo",
		"bar",
	}

	ref := config.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V([]config.Value{
		config.V("foo"),
		config.V("bar"),
	}), nv)
}

func TestFromTypedSliceNonEmptyRetainLocationIfUnchanged(t *testing.T) {
	var src = []string{
		"foo",
		"bar",
	}

	ref := config.V([]config.Value{
		config.NewValue("foo", config.Location{File: "foo"}),
		config.NewValue("baz", config.Location{File: "baz"}),
	})

	nv, err := FromTyped(src, ref)
	require.NoError(t, err)

	// Assert foo has retained its location.
	assert.Equal(t, config.NewValue("foo", config.Location{File: "foo"}), nv.Index(0))

	// Assert bar lost its location (because it was overwritten).
	assert.Equal(t, config.NewValue("bar", config.Location{}), nv.Index(1))
}

func TestFromTypedStringEmpty(t *testing.T) {
	var src string
	var ref = config.V("string")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NilValue, nv)
}

func TestFromTypedStringNonEmpty(t *testing.T) {
	var src = "new"
	var ref = config.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V("new"), nv)
}

func TestFromTypedStringNonEmptyOverwrite(t *testing.T) {
	var src = "new"
	var ref = config.V("old")
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V("new"), nv)
}

func TestFromTypedStringRetainsLocationsIfUnchanged(t *testing.T) {
	var src = "foo"
	var ref = config.NewValue("foo", config.Location{File: "foo"})
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NewValue("foo", config.Location{File: "foo"}), nv)
}

func TestFromTypedBoolEmpty(t *testing.T) {
	var src bool
	var ref = config.V(true)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NilValue, nv)
}

func TestFromTypedBoolNonEmpty(t *testing.T) {
	var src bool = true
	var ref = config.NilValue
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V(true), nv)
}

func TestFromTypedBoolNonEmptyOverwrite(t *testing.T) {
	var src bool = true
	var ref = config.V(false)
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.V(true), nv)
}

func TestFromTypedBoolRetainsLocationsIfUnchanged(t *testing.T) {
	var src = true
	var ref = config.NewValue(true, config.Location{File: "foo"})
	nv, err := FromTyped(src, ref)
	require.NoError(t, err)
	assert.Equal(t, config.NewValue(true, config.Location{File: "foo"}), nv)
}
