package dyn_test

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestInvalidValue(t *testing.T) {
	// Assert that the zero value of [dyn.Value] is the invalid value.
	var zero dyn.Value
	assert.Equal(t, zero, dyn.InvalidValue)
}

func TestValueIsAnchor(t *testing.T) {
	var zero dyn.Value
	assert.False(t, zero.IsAnchor())
	mark := zero.MarkAnchor()
	assert.True(t, mark.IsAnchor())
}

func TestValueAsMap(t *testing.T) {
	var zeroValue dyn.Value
	m, ok := zeroValue.AsMap()
	assert.False(t, ok)
	assert.Nil(t, m)

	var intValue = dyn.NewValue(1, dyn.Location{})
	m, ok = intValue.AsMap()
	assert.False(t, ok)
	assert.Nil(t, m)

	var mapValue = dyn.NewValue(
		map[string]dyn.Value{
			"key": dyn.NewValue("value", dyn.Location{File: "file", Line: 1, Column: 2}),
		},
		dyn.Location{File: "file", Line: 1, Column: 2},
	)
	m, ok = mapValue.AsMap()
	assert.True(t, ok)
	assert.Len(t, m, 1)
}

func TestValueIsValid(t *testing.T) {
	var zeroValue dyn.Value
	assert.False(t, zeroValue.IsValid())
	var intValue = dyn.NewValue(1, dyn.Location{})
	assert.True(t, intValue.IsValid())
}

func TestMarshalYAMLNilValue(t *testing.T) {
	var nilValue = dyn.NilValue
	v, err := nilValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "null", v.(*yaml.Node).Value)
}

func TestMarshalYAMLIntValue(t *testing.T) {
	var intValue = dyn.NewValue(1, dyn.Location{})
	v, err := intValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "1", v.(*yaml.Node).Value)
	assert.Equal(t, yaml.ScalarNode, v.(*yaml.Node).Kind)
}

func TestMarshalYAMLFloatValue(t *testing.T) {
	var floatValue = dyn.NewValue(1.0, dyn.Location{})
	v, err := floatValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "1", v.(*yaml.Node).Value)
	assert.Equal(t, yaml.ScalarNode, v.(*yaml.Node).Kind)
}

func TestMarshalYAMLBoolValue(t *testing.T) {
	var boolValue = dyn.NewValue(true, dyn.Location{})
	v, err := boolValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "true", v.(*yaml.Node).Value)
	assert.Equal(t, yaml.ScalarNode, v.(*yaml.Node).Kind)
}

func TestMarshalYAMLTimeValue(t *testing.T) {
	var timeValue = dyn.NewValue(time.Unix(0, 0), dyn.Location{})
	v, err := timeValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "1970-01-01 00:00:00 +0000 UTC", v.(*yaml.Node).Value)
	assert.Equal(t, yaml.ScalarNode, v.(*yaml.Node).Kind)
}

func TestMarshalYAMLSequenceValue(t *testing.T) {
	var sequenceValue = dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue("value1", dyn.Location{File: "file", Line: 1, Column: 2}),
			dyn.NewValue("value2", dyn.Location{File: "file", Line: 2, Column: 2}),
		},
		dyn.Location{File: "file", Line: 1, Column: 2},
	)
	v, err := sequenceValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, yaml.SequenceNode, v.(*yaml.Node).Kind)
	assert.Equal(t, "value1", v.(*yaml.Node).Content[0].Value)
	assert.Equal(t, "value2", v.(*yaml.Node).Content[1].Value)
}

func TestMarshalYAMLStringValue(t *testing.T) {
	var stringValue = dyn.NewValue("value", dyn.Location{})
	v, err := stringValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "value", v.(*yaml.Node).Value)
	assert.Equal(t, yaml.ScalarNode, v.(*yaml.Node).Kind)
}

func TestMarshalYAMLMapValue(t *testing.T) {
	var mapValue = dyn.NewValue(
		map[string]dyn.Value{
			"key3": dyn.NewValue("value3", dyn.Location{File: "file", Line: 3, Column: 2}),
			"key2": dyn.NewValue("value2", dyn.Location{File: "file", Line: 2, Column: 2}),
			"key1": dyn.NewValue("value1", dyn.Location{File: "file", Line: 1, Column: 2}),
		},
		dyn.Location{File: "file", Line: 1, Column: 2},
	)
	v, err := mapValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, v.(*yaml.Node).Kind)
	assert.Equal(t, "key1", v.(*yaml.Node).Content[0].Value)
	assert.Equal(t, "value1", v.(*yaml.Node).Content[1].Value)

	assert.Equal(t, "key2", v.(*yaml.Node).Content[2].Value)
	assert.Equal(t, "value2", v.(*yaml.Node).Content[3].Value)

	assert.Equal(t, "key3", v.(*yaml.Node).Content[4].Value)
	assert.Equal(t, "value3", v.(*yaml.Node).Content[5].Value)
}

func TestMarshalYAMLNestedValues(t *testing.T) {
	var mapValue = dyn.NewValue(
		map[string]dyn.Value{
			"key1": dyn.NewValue(
				map[string]dyn.Value{
					"key2": dyn.NewValue("value", dyn.Location{File: "file", Line: 1, Column: 2}),
				},
				dyn.Location{File: "file", Line: 1, Column: 2},
			),
		},
		dyn.Location{File: "file", Line: 1, Column: 2},
	)
	v, err := mapValue.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, v.(*yaml.Node).Kind)
	assert.Equal(t, "key1", v.(*yaml.Node).Content[0].Value)
	assert.Equal(t, yaml.MappingNode, v.(*yaml.Node).Content[1].Kind)
	assert.Equal(t, "key2", v.(*yaml.Node).Content[1].Content[0].Value)
	assert.Equal(t, "value", v.(*yaml.Node).Content[1].Content[1].Value)
}
