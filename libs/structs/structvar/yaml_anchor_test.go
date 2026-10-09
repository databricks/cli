package structvar_test

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

func loc(file string, line, column int) []diag.Location {
	return []diag.Location{{File: file, Line: line, Column: column}}
}

func TestYAMLAnchor01(t *testing.T) {
	file := "testdata/yaml/anchor_01.yml"
	v, sv := loadYAML(t, file)

	// The anchor is not reported as an unknown field, the other keys are.
	type known struct{}
	warnings := decodeFields[known](t, file)
	assert.NotContains(t, warnings, "unknown field: defaults")
	assert.Contains(t, warnings, "unknown field: shirt1")
	assert.Contains(t, warnings, "unknown field: shirt2")

	assert.Equal(t, "striped", at(t, v, "shirt1", "pattern"))
	assert.Equal(t, loc(file, 8, 12), locsAt(sv, "shirt1", "pattern"))
}

func TestYAMLAnchor02(t *testing.T) {
	file := "testdata/yaml/anchor_02.yml"
	v, sv := loadYAML(t, file)

	assert.Equal(t, "red", at(t, v, "shirt", "color"))
	assert.Equal(t, loc(file, 4, 10), locsAt(sv, "shirt", "color"))

	assert.Equal(t, "cotton", at(t, v, "shirt", "primary"))
	assert.Equal(t, loc(file, 8, 12), locsAt(sv, "shirt", "primary"))

	assert.Equal(t, "striped", at(t, v, "shirt", "pattern"))
	assert.Equal(t, loc(file, 13, 12), locsAt(sv, "shirt", "pattern"))
}

func TestYAMLAnchor03(t *testing.T) {
	file := "testdata/yaml/anchor_03.yml"
	v, sv := loadYAML(t, file)

	// Assert the override took place.
	assert.Equal(t, "blue", at(t, v, "shirt", "color"))
	assert.Equal(t, loc(file, 10, 10), locsAt(sv, "shirt", "color"))
}

func TestYAMLAnchor04(t *testing.T) {
	file := "testdata/yaml/anchor_04.yml"
	v, sv := loadYAML(t, file)

	assert.Equal(t, "San Francisco", at(t, v, "person1", "address", "city"))
	assert.Equal(t, loc(file, 4, 9), locsAt(sv, "person1", "address", "city"))

	assert.Equal(t, "Los Angeles", at(t, v, "person2", "address", "city"))
	assert.Equal(t, loc(file, 16, 11), locsAt(sv, "person2", "address", "city"))
}

func TestYAMLAnchor05(t *testing.T) {
	file := "testdata/yaml/anchor_05.yml"
	v, sv := loadYAML(t, file)

	assert.Equal(t, "wifi", at(t, v, "phone1", "features", 0))
	assert.Equal(t, loc(file, 4, 5), locsAt(sv, "phone1", "features", 0))
	assert.Equal(t, "bluetooth", at(t, v, "phone1", "features", 1))
	assert.Equal(t, loc(file, 5, 5), locsAt(sv, "phone1", "features", 1))
}

func TestYAMLAnchor06(t *testing.T) {
	file := "testdata/yaml/anchor_06.yml"
	v, sv := loadYAML(t, file)

	assert.Equal(t, "Hello, World!", at(t, v, "greeting1"))
	assert.Equal(t, loc(file, 2, 16), locsAt(sv, "greeting1"))
}

func TestYAMLAnchor07(t *testing.T) {
	file := "testdata/yaml/anchor_07.yml"
	v, sv := loadYAML(t, file)

	assert.Equal(t, "Alice", at(t, v, "person1", "name"))
	assert.Equal(t, loc(file, 5, 9), locsAt(sv, "person1", "name"))

	assert.Equal(t, 25, at(t, v, "person1", "age"))
	assert.Equal(t, loc(file, 2, 13), locsAt(sv, "person1", "age"))
}

func TestYAMLAnchor08(t *testing.T) {
	file := "testdata/yaml/anchor_08.yml"
	v, sv := loadYAML(t, file)

	assert.Equal(t, "user1", at(t, v, "user1", "username"))
	assert.Equal(t, loc(file, 5, 13), locsAt(sv, "user1", "username"))

	assert.Equal(t, true, at(t, v, "user1", "active"))
	assert.Equal(t, loc(file, 2, 11), locsAt(sv, "user1", "active"))
}

func TestYAMLAnchor09(t *testing.T) {
	file := "testdata/yaml/anchor_09.yml"
	_, _, err := decodeFile(t, file)
	assert.ErrorContains(t, err, `cyclic reference to anchor "x"`)
}

func TestYAMLAnchor10(t *testing.T) {
	file := "testdata/yaml/anchor_10.yml"
	v, _ := loadYAML(t, file)

	assert.Equal(t, 1, at(t, v, "use1", "value"))
	assert.Equal(t, 1, at(t, v, "use2", "value"))
}
