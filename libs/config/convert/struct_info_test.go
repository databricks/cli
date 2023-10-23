package convert

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
