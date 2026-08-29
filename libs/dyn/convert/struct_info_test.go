package convert

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	si := getStructInfo(reflect.TypeFor[Tmp]())
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

	si := getStructInfo(reflect.TypeFor[Tmp]())
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

	si := getStructInfo(reflect.TypeFor[Tmp]())
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

	si := getStructInfo(reflect.TypeFor[Tmp]())
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

	si := getStructInfo(reflect.TypeFor[Tmp]())
	fv := si.FieldValues(reflect.ValueOf(src))
	assert.Len(t, fv, 2)
	assert.Equal(t, "foo", fv[0].Key)
	assert.True(t, reflect.ValueOf("foo").Equal(fv[0].Value))
	assert.Equal(t, "bar", fv[1].Key)
	assert.True(t, reflect.ValueOf("bar").Equal(fv[1].Value))
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

	si := getStructInfo(reflect.TypeFor[Tmp]())
	fv := si.FieldValues(reflect.ValueOf(src))
	assert.Len(t, fv, 2)
	assert.Equal(t, "foo", fv[0].Key)
	assert.Equal(t, "bar", fv[1].Key)
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

		si := getStructInfo(reflect.TypeFor[Tmp]())
		fv := si.FieldValues(reflect.ValueOf(src))
		assert.Len(t, fv, 2)
		assert.Equal(t, "foo", fv[0].Key)
		assert.Equal(t, "bar", fv[1].Key)
	})

	// Test that fields of embedded types are skipped if the embedded type is nil.
	t.Run("top level is set", func(t *testing.T) {
		src := Tmp{
			Foo: &Foo{
				Foo: "foo",
				Bar: nil,
			},
		}

		si := getStructInfo(reflect.TypeFor[Tmp]())
		fv := si.FieldValues(reflect.ValueOf(src))
		assert.Len(t, fv, 1)
		assert.Equal(t, "foo", fv[0].Key)
	})

	// Test that fields of embedded types are skipped if the embedded type is nil.
	t.Run("none are set", func(t *testing.T) {
		src := Tmp{
			Foo: nil,
		}

		si := getStructInfo(reflect.TypeFor[Tmp]())
		fv := si.FieldValues(reflect.ValueOf(src))
		assert.Empty(t, fv)
	})
}

func TestStructInfoValueFieldAbsent(t *testing.T) {
	type Tmp struct {
		Foo string `json:"foo"`
	}

	si := getStructInfo(reflect.TypeFor[Tmp]())
	assert.Nil(t, si.ValueField)
}

func TestStructInfoValueFieldPresent(t *testing.T) {
	type Tmp struct {
		Foo dyn.Value
	}

	si := getStructInfo(reflect.TypeFor[Tmp]())
	assert.NotNil(t, si.ValueField)
}

func TestStructInfoValueFieldMultiple(t *testing.T) {
	type Tmp struct {
		Foo dyn.Value
		Bar dyn.Value
	}

	assert.Panics(t, func() {
		getStructInfo(reflect.TypeFor[Tmp]())
	})
}

func TestSensitiveFieldNamesPlain(t *testing.T) {
	type Tmp struct {
		Name  string `json:"name"`
		Token string `json:"token" bundle:"sensitive"`
	}

	fields := SensitiveFieldNames(reflect.TypeFor[Tmp]())
	assert.True(t, fields["token"])
	assert.False(t, fields["name"])
}

func TestSensitiveFieldNamesPointerDereference(t *testing.T) {
	type Tmp struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	fields := SensitiveFieldNames(reflect.TypeFor[*Tmp]())
	assert.True(t, fields["token"])
}

func TestSensitiveFieldNamesNilForNonStruct(t *testing.T) {
	assert.Nil(t, SensitiveFieldNames(reflect.TypeFor[string]()))
}

func TestSensitiveFieldNamesNilWhenNone(t *testing.T) {
	type Tmp struct {
		Name string `json:"name"`
	}

	assert.Nil(t, SensitiveFieldNames(reflect.TypeFor[Tmp]()))
}

func TestSensitiveFieldNamesEmbeddedByValue(t *testing.T) {
	type Inner struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	type Outer struct {
		Name string `json:"name"`
		Inner
	}

	fields := SensitiveFieldNames(reflect.TypeFor[Outer]())
	assert.True(t, fields["token"])
	assert.False(t, fields["name"])
}

func TestSensitiveFieldNamesEmbeddedByPointer(t *testing.T) {
	type Inner struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	type Outer struct {
		Name string `json:"name"`
		*Inner
	}

	fields := SensitiveFieldNames(reflect.TypeFor[Outer]())
	assert.True(t, fields["token"])
	assert.False(t, fields["name"])
}

func TestSensitiveFieldNamesTopLevelPrecedence(t *testing.T) {
	// A sensitive field in an embedded struct is shadowed by a non-sensitive
	// field of the same JSON name at the top level — top level wins.
	type Inner struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	type Outer struct {
		Token string `json:"token"` // not sensitive; shadows Inner.Token
		Inner
	}

	fields := SensitiveFieldNames(reflect.TypeFor[Outer]())
	assert.False(t, fields["token"])
}

// TestSensitiveFieldNamesUsage demonstrates using SensitiveFieldNames with
// FromTyped: a sensitive field's value round-trips through dyn.Value and the
// caller can check which fields to mask before marshaling.
func TestSensitiveFieldNamesUsage(t *testing.T) {
	type Resource struct {
		Name  string `json:"name"`
		Token string `json:"token" bundle:"sensitive"`
	}

	src := Resource{Name: "my-resource", Token: "s3cr3t"}
	v, err := FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	fields := SensitiveFieldNames(reflect.TypeFor[Resource]())
	assert.True(t, fields["token"], "token should be identified as sensitive")

	// Verify the value round-tripped correctly before masking.
	tok, err := dyn.GetByPath(v, dyn.NewPath(dyn.Key("token")))
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", tok.MustString())
}
