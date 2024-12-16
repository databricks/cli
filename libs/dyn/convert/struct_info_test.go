package convert

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestStructInfoPlain(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar,omitempty"`

		// Baz must be skipped.
		Baz string `json:""`

		// Qux must be skipped.
		Qux string `json:"-"`
	}

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	assert.Len(t, si.Fields, 2)
	assert.Equal(t, []int{0}, si.Fields["foo"])
	assert.Equal(t, []int{1}, si.Fields["bar"])
}

func TestStructInfoAnonymousByValue(t *testing.T) {
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

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	assert.Len(t, si.Fields, 2)
	assert.Equal(t, []int{0, 0}, si.Fields["foo"])
	assert.Equal(t, []int{0, 1, 0}, si.Fields["bar"])
}

func TestStructInfoAnonymousByValuePrecedence(t *testing.T) {
	type Bar struct {
		Bar string `json:"bar"`
	}

	type Foo struct {
		Foo string `json:"foo"`
		Bar
	}

	type Tmp struct {
		// "foo" comes from [Foo].
		Foo
		// "bar" comes from [Bar] directly, not through [Foo].
		Bar
	}

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	assert.Len(t, si.Fields, 2)
	assert.Equal(t, []int{0, 0}, si.Fields["foo"])
	assert.Equal(t, []int{1, 0}, si.Fields["bar"])
}

func TestStructInfoAnonymousByPointer(t *testing.T) {
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

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	assert.Len(t, si.Fields, 2)
	assert.Equal(t, []int{0, 0}, si.Fields["foo"])
	assert.Equal(t, []int{0, 1, 0}, si.Fields["bar"])
}

func TestStructInfoFieldValues(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
		Bar string `json:"bar"`
	}

	src := Tmp{
		Foo: "foo",
		Bar: "bar",
	}

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	fv := si.FieldValues(reflect.ValueOf(src))
	assert.Len(t, fv, 2)
	assert.Equal(t, "foo", fv["foo"].String())
	assert.Equal(t, "bar", fv["bar"].String())
}

func TestStructInfoFieldValuesAnonymousByValue(t *testing.T) {
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

	src := Tmp{
		Foo: Foo{
			Foo: "foo",
			Bar: Bar{
				Bar: "bar",
			},
		},
	}

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	fv := si.FieldValues(reflect.ValueOf(src))
	assert.Len(t, fv, 2)
	assert.Equal(t, "foo", fv["foo"].String())
	assert.Equal(t, "bar", fv["bar"].String())
}

func TestStructInfoFieldValuesAnonymousByPointer(t *testing.T) {
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

	// Test that the embedded fields are dereferenced properly.
	t.Run("all are set", func(t *testing.T) {
		src := Tmp{
			Foo: &Foo{
				Foo: "foo",
				Bar: &Bar{
					Bar: "bar",
				},
			},
		}

		si := getStructInfo(reflect.TypeOf(Tmp{}))
		fv := si.FieldValues(reflect.ValueOf(src))
		assert.Len(t, fv, 2)
		assert.Equal(t, "foo", fv["foo"].String())
		assert.Equal(t, "bar", fv["bar"].String())
	})

	// Test that fields of embedded types are skipped if the embedded type is nil.
	t.Run("top level is set", func(t *testing.T) {
		src := Tmp{
			Foo: &Foo{
				Foo: "foo",
				Bar: nil,
			},
		}

		si := getStructInfo(reflect.TypeOf(Tmp{}))
		fv := si.FieldValues(reflect.ValueOf(src))
		assert.Len(t, fv, 1)
		assert.Equal(t, "foo", fv["foo"].String())
	})

	// Test that fields of embedded types are skipped if the embedded type is nil.
	t.Run("none are set", func(t *testing.T) {
		src := Tmp{
			Foo: nil,
		}

		si := getStructInfo(reflect.TypeOf(Tmp{}))
		fv := si.FieldValues(reflect.ValueOf(src))
		assert.Empty(t, fv)
	})
}

func TestStructInfoValueFieldAbsent(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	assert.Nil(t, si.ValueField)
}

func TestStructInfoValueFieldPresent(t *testing.T) {
	type Tmp struct {
		Foo dyn.Value
	}

	si := getStructInfo(reflect.TypeOf(Tmp{}))
	assert.NotNil(t, si.ValueField)
}

func TestStructInfoValueFieldMultiple(t *testing.T) {
	type Tmp struct {
		Foo dyn.Value
		Bar dyn.Value
	}

	assert.Panics(t, func() {
		getStructInfo(reflect.TypeOf(Tmp{}))
	})
}
