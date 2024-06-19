package yamlloader_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestYAMLAnchor01(t *testing.T) {
	file := "testdata/anchor_01.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	assert.True(t, self.Get("defaults").IsAnchor())
	assert.False(t, self.Get("shirt1").IsAnchor())
	assert.False(t, self.Get("shirt2").IsAnchor())

	pattern := self.Get("shirt1").Get("pattern")
	assert.Equal(t, "striped", pattern.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 8, Column: 12}, pattern.Location())
}

func TestYAMLAnchor02(t *testing.T) {
	file := "testdata/anchor_02.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	color := self.Get("shirt").Get("color")
	assert.Equal(t, "red", color.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 4, Column: 10}, color.Location())

	primary := self.Get("shirt").Get("primary")
	assert.Equal(t, "cotton", primary.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 8, Column: 12}, primary.Location())

	pattern := self.Get("shirt").Get("pattern")
	assert.Equal(t, "striped", pattern.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 13, Column: 12}, pattern.Location())
}

func TestYAMLAnchor03(t *testing.T) {
	file := "testdata/anchor_03.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	// Assert the override took place.
	blue := self.Get("shirt").Get("color")
	assert.Equal(t, "blue", blue.AsAny())
	assert.Equal(t, file, blue.Location().File)
	assert.Equal(t, 10, blue.Location().Line)
	assert.Equal(t, 10, blue.Location().Column)
}

func TestYAMLAnchor04(t *testing.T) {
	file := "testdata/anchor_04.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	p1 := self.Get("person1").Get("address").Get("city")
	assert.Equal(t, "San Francisco", p1.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 4, Column: 9}, p1.Location())

	p2 := self.Get("person2").Get("address").Get("city")
	assert.Equal(t, "Los Angeles", p2.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 16, Column: 11}, p2.Location())
}

func TestYAMLAnchor05(t *testing.T) {
	file := "testdata/anchor_05.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	features := self.Get("phone1").Get("features")
	assert.Equal(t, "wifi", features.Index(0).AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 4, Column: 5}, features.Index(0).Location())
	assert.Equal(t, "bluetooth", features.Index(1).AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 5, Column: 5}, features.Index(1).Location())
}

func TestYAMLAnchor06(t *testing.T) {
	file := "testdata/anchor_06.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	greeting := self.Get("greeting1")
	assert.Equal(t, "Hello, World!", greeting.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 2, Column: 16}, greeting.Location())
}

func TestYAMLAnchor07(t *testing.T) {
	file := "testdata/anchor_07.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	name := self.Get("person1").Get("name")
	assert.Equal(t, "Alice", name.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 5, Column: 9}, name.Location())

	age := self.Get("person1").Get("age")
	assert.Equal(t, 25, age.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 2, Column: 13}, age.Location())
}

func TestYAMLAnchor08(t *testing.T) {
	file := "testdata/anchor_08.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	username := self.Get("user1").Get("username")
	assert.Equal(t, "user1", username.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 5, Column: 13}, username.Location())

	active := self.Get("user1").Get("active")
	assert.Equal(t, true, active.AsAny())
	assert.Equal(t, dyn.Location{File: file, Line: 2, Column: 11}, active.Location())
}
