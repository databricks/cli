package yamlloader_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynassert"
)

func TestYAMLAnchor01(t *testing.T) {
	file := "testdata/anchor_01.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	dynassert.True(t, self.Get("defaults").IsAnchor())
	dynassert.False(t, self.Get("shirt1").IsAnchor())
	dynassert.False(t, self.Get("shirt2").IsAnchor())

	pattern := self.Get("shirt1").Get("pattern")
	dynassert.Equal(t, "striped", pattern.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 8, Column: 12}, pattern.Location())
}

func TestYAMLAnchor02(t *testing.T) {
	file := "testdata/anchor_02.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	color := self.Get("shirt").Get("color")
	dynassert.Equal(t, "red", color.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 4, Column: 10}, color.Location())

	primary := self.Get("shirt").Get("primary")
	dynassert.Equal(t, "cotton", primary.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 8, Column: 12}, primary.Location())

	pattern := self.Get("shirt").Get("pattern")
	dynassert.Equal(t, "striped", pattern.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 13, Column: 12}, pattern.Location())
}

func TestYAMLAnchor03(t *testing.T) {
	file := "testdata/anchor_03.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	// Assert the override took place.
	blue := self.Get("shirt").Get("color")
	dynassert.Equal(t, "blue", blue.AsAny())
	dynassert.Equal(t, file, blue.Location().File)
	dynassert.Equal(t, 10, blue.Location().Line)
	dynassert.Equal(t, 10, blue.Location().Column)
}

func TestYAMLAnchor04(t *testing.T) {
	file := "testdata/anchor_04.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	p1 := self.Get("person1").Get("address").Get("city")
	dynassert.Equal(t, "San Francisco", p1.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 4, Column: 9}, p1.Location())

	p2 := self.Get("person2").Get("address").Get("city")
	dynassert.Equal(t, "Los Angeles", p2.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 16, Column: 11}, p2.Location())
}

func TestYAMLAnchor05(t *testing.T) {
	file := "testdata/anchor_05.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	features := self.Get("phone1").Get("features")
	dynassert.Equal(t, "wifi", features.Index(0).AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 4, Column: 5}, features.Index(0).Location())
	dynassert.Equal(t, "bluetooth", features.Index(1).AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 5, Column: 5}, features.Index(1).Location())
}

func TestYAMLAnchor06(t *testing.T) {
	file := "testdata/anchor_06.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	greeting := self.Get("greeting1")
	dynassert.Equal(t, "Hello, World!", greeting.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 2, Column: 16}, greeting.Location())
}

func TestYAMLAnchor07(t *testing.T) {
	file := "testdata/anchor_07.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	name := self.Get("person1").Get("name")
	dynassert.Equal(t, "Alice", name.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 5, Column: 9}, name.Location())

	age := self.Get("person1").Get("age")
	dynassert.Equal(t, 25, age.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 2, Column: 13}, age.Location())
}

func TestYAMLAnchor08(t *testing.T) {
	file := "testdata/anchor_08.yml"
	self := loadYAML(t, file)
	dynassert.NotEqual(t, dyn.NilValue, self)

	username := self.Get("user1").Get("username")
	dynassert.Equal(t, "user1", username.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 5, Column: 13}, username.Location())

	active := self.Get("user1").Get("active")
	dynassert.Equal(t, true, active.AsAny())
	dynassert.Equal(t, dyn.Location{File: file, Line: 2, Column: 11}, active.Location())
}
