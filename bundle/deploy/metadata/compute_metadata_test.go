package metadata

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: test primitive pointers

func TestComputeMetadataWalk(t *testing.T) {
	type foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}

	config := foo{
		privateField:       2,
		IgnoredForMetadata: "abc",
		IncludedInMetadata: "xyz",
	}
	metadata := foo{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, foo{IncludedInMetadata: "xyz"}, metadata)
}

func TestComputeMetadataRecursivelyWalkAnonymousStruct(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Foo: Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Foo: Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataRecursivelyWalkAnonymousStructPointer(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		*Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Foo: &Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Foo: &Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataRecursivelyWalkStruct(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		Mango Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Mango: Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Mango: Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}

func TestComputeMetadataRecursivelyWalkStructPointer(t *testing.T) {
	type Foo struct {
		privateField       int `bundle:"metadata"`
		IgnoredForMetadata string
		IncludedInMetadata string `bundle:"metadata"`
	}
	type bar struct {
		Apple int `bundle:"readonly,metadata"`
		Mango *Foo
		Guava string
	}

	config := bar{
		Apple: 123,
		Mango: &Foo{
			privateField:       2,
			IgnoredForMetadata: "abc",
			IncludedInMetadata: "xyz",
		},
		Guava: "guava",
	}

	metadata := bar{}
	err := walk(reflect.ValueOf(config), reflect.ValueOf(&metadata).Elem())
	assert.NoError(t, err)
	assert.Equal(t, bar{
		Mango: &Foo{
			IncludedInMetadata: "xyz",
		},
		Apple: 123,
	}, metadata)
}
