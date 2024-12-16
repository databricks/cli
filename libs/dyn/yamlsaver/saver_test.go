package yamlsaver

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestMarshalNilValue(t *testing.T) {
	s := NewSaver()
	nilValue := dyn.NilValue
	v, err := s.toYamlNode(nilValue)
	assert.NoError(t, err)
	assert.Equal(t, "null", v.Value)
}

func TestMarshalIntValue(t *testing.T) {
	s := NewSaver()
	intValue := dyn.V(1)
	v, err := s.toYamlNode(intValue)
	assert.NoError(t, err)
	assert.Equal(t, "1", v.Value)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalFloatValue(t *testing.T) {
	s := NewSaver()
	floatValue := dyn.V(1.0)
	v, err := s.toYamlNode(floatValue)
	assert.NoError(t, err)
	assert.Equal(t, "1", v.Value)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalBoolValue(t *testing.T) {
	s := NewSaver()
	boolValue := dyn.V(true)
	v, err := s.toYamlNode(boolValue)
	assert.NoError(t, err)
	assert.Equal(t, "true", v.Value)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalTimeValue(t *testing.T) {
	tm, err := dyn.NewTime("1970-01-01")
	require.NoError(t, err)

	s := NewSaver()
	timeValue := dyn.V(tm)
	v, err := s.toYamlNode(timeValue)
	assert.NoError(t, err)
	assert.Equal(t, "1970-01-01", v.Value)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalSequenceValue(t *testing.T) {
	s := NewSaver()
	sequenceValue := dyn.NewValue(
		[]dyn.Value{
			dyn.NewValue("value1", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
			dyn.NewValue("value2", []dyn.Location{{File: "file", Line: 2, Column: 2}}),
		},
		[]dyn.Location{{File: "file", Line: 1, Column: 2}},
	)
	v, err := s.toYamlNode(sequenceValue)
	assert.NoError(t, err)
	assert.Equal(t, yaml.SequenceNode, v.Kind)
	assert.Equal(t, "value1", v.Content[0].Value)
	assert.Equal(t, "value2", v.Content[1].Value)
}

func TestMarshalStringValue(t *testing.T) {
	s := NewSaver()
	stringValue := dyn.V("value")
	v, err := s.toYamlNode(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, "value", v.Value)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalMapValue(t *testing.T) {
	s := NewSaver()
	mapValue := dyn.NewValue(
		map[string]dyn.Value{
			"key3": dyn.NewValue("value3", []dyn.Location{{File: "file", Line: 3, Column: 2}}),
			"key2": dyn.NewValue("value2", []dyn.Location{{File: "file", Line: 2, Column: 2}}),
			"key1": dyn.NewValue("value1", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
		},
		[]dyn.Location{{File: "file", Line: 1, Column: 2}},
	)

	v, err := s.toYamlNode(mapValue)
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, v.Kind)
	assert.Equal(t, "key1", v.Content[0].Value)
	assert.Equal(t, "value1", v.Content[1].Value)

	assert.Equal(t, "key2", v.Content[2].Value)
	assert.Equal(t, "value2", v.Content[3].Value)

	assert.Equal(t, "key3", v.Content[4].Value)
	assert.Equal(t, "value3", v.Content[5].Value)
}

func TestMarshalNestedValues(t *testing.T) {
	s := NewSaver()
	mapValue := dyn.NewValue(
		map[string]dyn.Value{
			"key1": dyn.NewValue(
				map[string]dyn.Value{
					"key2": dyn.NewValue("value", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
				},
				[]dyn.Location{{File: "file", Line: 1, Column: 2}},
			),
		},
		[]dyn.Location{{File: "file", Line: 1, Column: 2}},
	)
	v, err := s.toYamlNode(mapValue)
	assert.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, v.Kind)
	assert.Equal(t, "key1", v.Content[0].Value)
	assert.Equal(t, yaml.MappingNode, v.Content[1].Kind)
	assert.Equal(t, "key2", v.Content[1].Content[0].Value)
	assert.Equal(t, "value", v.Content[1].Content[1].Value)
}

func TestMarshalHexadecimalValueIsQuoted(t *testing.T) {
	s := NewSaver()
	hexValue := dyn.V(0x123)
	v, err := s.toYamlNode(hexValue)
	assert.NoError(t, err)
	assert.Equal(t, "291", v.Value)
	assert.Equal(t, yaml.Style(0), v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)

	stringValue := dyn.V("0x123")
	v, err = s.toYamlNode(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, "0x123", v.Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalBinaryValueIsQuoted(t *testing.T) {
	s := NewSaver()
	binaryValue := dyn.V(0b101)
	v, err := s.toYamlNode(binaryValue)
	assert.NoError(t, err)
	assert.Equal(t, "5", v.Value)
	assert.Equal(t, yaml.Style(0), v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)

	stringValue := dyn.V("0b101")
	v, err = s.toYamlNode(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, "0b101", v.Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalOctalValueIsQuoted(t *testing.T) {
	s := NewSaver()
	octalValue := dyn.V(0o123)
	v, err := s.toYamlNode(octalValue)
	assert.NoError(t, err)
	assert.Equal(t, "83", v.Value)
	assert.Equal(t, yaml.Style(0), v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)

	stringValue := dyn.V("0123")
	v, err = s.toYamlNode(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, "0123", v.Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalFloatValueIsQuoted(t *testing.T) {
	s := NewSaver()
	floatValue := dyn.V(1.0)
	v, err := s.toYamlNode(floatValue)
	assert.NoError(t, err)
	assert.Equal(t, "1", v.Value)
	assert.Equal(t, yaml.Style(0), v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)

	stringValue := dyn.V("1.0")
	v, err = s.toYamlNode(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, "1.0", v.Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestMarshalBoolValueIsQuoted(t *testing.T) {
	s := NewSaver()
	boolValue := dyn.V(true)
	v, err := s.toYamlNode(boolValue)
	assert.NoError(t, err)
	assert.Equal(t, "true", v.Value)
	assert.Equal(t, yaml.Style(0), v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)

	stringValue := dyn.V("true")
	v, err = s.toYamlNode(stringValue)
	assert.NoError(t, err)
	assert.Equal(t, "true", v.Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Style)
	assert.Equal(t, yaml.ScalarNode, v.Kind)
}

func TestCustomStylingWithNestedMap(t *testing.T) {
	s := NewSaverWithStyle(map[string]yaml.Style{
		"styled": yaml.DoubleQuotedStyle,
	})

	styledMap := dyn.NewValue(
		map[string]dyn.Value{
			"key1": dyn.NewValue("value1", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
			"key2": dyn.NewValue("value2", []dyn.Location{{File: "file", Line: 2, Column: 2}}),
		},
		[]dyn.Location{{File: "file", Line: -2, Column: 2}},
	)

	unstyledMap := dyn.NewValue(
		map[string]dyn.Value{
			"key3": dyn.NewValue("value3", []dyn.Location{{File: "file", Line: 1, Column: 2}}),
			"key4": dyn.NewValue("value4", []dyn.Location{{File: "file", Line: 2, Column: 2}}),
		},
		[]dyn.Location{{File: "file", Line: -1, Column: 2}},
	)

	val := dyn.NewValue(
		map[string]dyn.Value{
			"styled":   styledMap,
			"unstyled": unstyledMap,
		},
		[]dyn.Location{{File: "file", Line: 1, Column: 2}},
	)

	mv, err := s.toYamlNode(val)
	assert.NoError(t, err)

	// Check that the styled map is quoted
	v := mv.Content[1]

	assert.Equal(t, yaml.MappingNode, v.Kind)
	assert.Equal(t, "key1", v.Content[0].Value)
	assert.Equal(t, "value1", v.Content[1].Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Content[0].Style)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Content[1].Style)

	assert.Equal(t, "key2", v.Content[2].Value)
	assert.Equal(t, "value2", v.Content[3].Value)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Content[2].Style)
	assert.Equal(t, yaml.DoubleQuotedStyle, v.Content[3].Style)

	// Check that the unstyled map is not quoted
	v = mv.Content[3]

	assert.Equal(t, yaml.MappingNode, v.Kind)
	assert.Equal(t, "key3", v.Content[0].Value)
	assert.Equal(t, "value3", v.Content[1].Value)
	assert.Equal(t, yaml.Style(0), v.Content[0].Style)
	assert.Equal(t, yaml.Style(0), v.Content[1].Style)

	assert.Equal(t, "key4", v.Content[2].Value)
	assert.Equal(t, "value4", v.Content[3].Value)
	assert.Equal(t, yaml.Style(0), v.Content[2].Style)
	assert.Equal(t, yaml.Style(0), v.Content[3].Style)
}
