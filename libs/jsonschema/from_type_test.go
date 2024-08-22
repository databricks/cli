package jsonschema

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/jsonschema/test_types"
	"github.com/stretchr/testify/assert"
)

func TestFromTypeBasic(t *testing.T) {
	type myStruct struct {
		S string      `json:"s"`
		I *int        `json:"i,omitempty"`
		V interface{} `json:"v,omitempty"`

		// These fields should be ignored in the resulting schema.
		NotAnnotated     string
		DashedTag        string `json:"-"`
		notExported      string `json:"not_exported"`
		InternalTagged   string `json:"internal_tagged" bundle:"internal"`
		DeprecatedTagged string `json:"deprecated_tagged" bundle:"deprecated"`
		ReadOnlyTagged   string `json:"readonly_tagged" bundle:"readonly"`
	}

	strRef := "#/$defs/string"
	boolRef := "#/$defs/bool"
	intRef := "#/$defs/int"
	interfaceRef := "#/$defs/interface"

	tcases := []struct {
		name     string
		typ      reflect.Type
		expected Schema
	}{
		{
			name: "int",
			typ:  reflect.TypeOf(int(0)),
			expected: Schema{
				Type: "integer",
			},
		},
		{
			name: "string",
			typ:  reflect.TypeOf(""),
			expected: Schema{
				Type: "string",
			},
		},
		{
			name: "bool",
			typ:  reflect.TypeOf(true),
			expected: Schema{
				Type: "boolean",
			},
		},
		{
			name: "float64",
			typ:  reflect.TypeOf(float64(0)),
			expected: Schema{
				Type: "number",
			},
		},
		{
			name: "struct",
			typ:  reflect.TypeOf(myStruct{}),
			expected: Schema{
				Type: "object",
				Definitions: map[string]any{
					"interface": Schema{
						Type: "null",
					},
					"string": Schema{
						Type: "string",
					},
					"int": Schema{
						Type: "integer",
					},
				},
				Properties: map[string]*Schema{
					"s": {
						Reference: &strRef,
					},
					"i": {
						Reference: &intRef,
					},
					"v": {
						Reference: &interfaceRef,
					},
				},
				AdditionalProperties: false,
				Required:             []string{"s"},
			},
		},
		{
			name: "slice",
			typ:  reflect.TypeOf([]bool{}),
			expected: Schema{
				Type: "array",
				Definitions: map[string]any{
					"bool": Schema{
						Type: "boolean",
					},
				},
				Items: &Schema{
					Reference: &boolRef,
				},
			},
		},
		{
			name: "map",
			typ:  reflect.TypeOf(map[string]int{}),
			expected: Schema{
				Type: "object",
				Definitions: map[string]any{
					"int": Schema{
						Type: "integer",
					},
				},
				AdditionalProperties: &Schema{
					Reference: &intRef,
				},
			},
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := FromType(tc.typ, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, s)

			// jsonSchema, err := json.MarshalIndent(s, "		", "	")
			// assert.NoError(t, err)

			// expectedJson, err := json.MarshalIndent(tc.expected, "		", "	")
			// assert.NoError(t, err)

			// t.Log("[DEBUG] actual: ", string(jsonSchema))
			// t.Log("[DEBUG] expected: ", string(expectedJson))
		})
	}
}

func TestGetStructFields(t *testing.T) {
	type EmbeddedStruct struct {
		I int
		B bool
	}

	type MyStruct struct {
		S string
		*EmbeddedStruct
	}

	fields := getStructFields(reflect.TypeOf(MyStruct{}))
	assert.Len(t, fields, 3)
	assert.Equal(t, "S", fields[0].Name)
	assert.Equal(t, "I", fields[1].Name)
	assert.Equal(t, "B", fields[2].Name)
}

func TestFromTypeNested(t *testing.T) {
	type Inner struct {
		S string `json:"s"`
	}

	type Outer struct {
		I     string `json:"i"`
		Inner Inner  `json:"inner"`
	}

	innerRef := "#/$defs/github.com/databricks/cli/libs/jsonschema.Inner"
	strRef := "#/$defs/string"

	expectedDefinitions := map[string]any{
		"github.com": map[string]any{
			"databricks": map[string]any{
				"cli": map[string]any{
					"libs": map[string]any{
						"jsonschema.Inner": Schema{
							Type: "object",
							Properties: map[string]*Schema{
								"s": {
									Reference: &strRef,
								},
							},
							AdditionalProperties: false,
							Required:             []string{"s"},
						},
					},
				},
			},
		},
		"string": Schema{
			Type: "string",
		},
	}

	tcases := []struct {
		name     string
		typ      reflect.Type
		expected Schema
	}{
		{
			name: "struct in struct",
			typ:  reflect.TypeOf(Outer{}),
			expected: Schema{
				Type:        "object",
				Definitions: expectedDefinitions,
				Properties: map[string]*Schema{
					"i": {
						Reference: &strRef,
					},
					"inner": {
						Reference: &innerRef,
					},
				},
				AdditionalProperties: false,
				Required:             []string{"i", "inner"},
			},
		},
		{
			name: "struct as a map value",
			typ:  reflect.TypeOf(map[string]Inner{}),
			expected: Schema{
				Type:        "object",
				Definitions: expectedDefinitions,
				AdditionalProperties: &Schema{
					Reference: &innerRef,
				},
			},
		},
		{
			name: "struct as a slice element",
			typ:  reflect.TypeOf([]Inner{}),
			expected: Schema{
				Type:        "array",
				Definitions: expectedDefinitions,
				Items: &Schema{
					Reference: &innerRef,
				},
			},
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := FromType(tc.typ, nil)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, s)
		})
	}
}

// TODO: Call out in the PR description that recursive Go types are supported.
func TestFromTypeRecursive(t *testing.T) {
	fooRef := "#/$defs/github.com/databricks/cli/libs/jsonschema/test_types.Foo"
	barRef := "#/$defs/github.com/databricks/cli/libs/jsonschema/test_types.Bar"

	expected := Schema{
		Type: "object",
		Definitions: map[string]any{
			"github.com": map[string]any{
				"databricks": map[string]any{
					"cli": map[string]any{
						"libs": map[string]any{
							"jsonschema": map[string]any{
								"test_types.Bar": Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"foo": {
											Reference: &fooRef,
										},
									},
									AdditionalProperties: false,
									Required:             []string{},
								},
								"test_types.Foo": Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"bar": {
											Reference: &barRef,
										},
									},
									AdditionalProperties: false,
									Required:             []string{},
								},
							},
						},
					},
				},
			},
		},
		Properties: map[string]*Schema{
			"foo": {
				Reference: &fooRef,
			},
		},
		AdditionalProperties: false,
		Required:             []string{"foo"},
	}

	s, err := FromType(reflect.TypeOf(test_types.Outer{}), nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, s)
}

func TestFromTypeSelfReferential(t *testing.T) {
	selfRef := "#/$defs/github.com/databricks/cli/libs/jsonschema/test_types.Self"
	stringRef := "#/$defs/string"

	expected := Schema{
		Type: "object",
		Definitions: map[string]any{
			"github.com": map[string]any{
				"databricks": map[string]any{
					"cli": map[string]any{
						"libs": map[string]any{
							"jsonschema": map[string]any{
								"test_types.Self": Schema{
									Type: "object",
									Properties: map[string]*Schema{
										"self": {
											Reference: &selfRef,
										},
										"s": {
											Reference: &stringRef,
										},
									},
									AdditionalProperties: false,
									Required:             []string{},
								},
							},
						},
					},
				},
			},
			"string": Schema{
				Type: "string",
			},
		},
		Properties: map[string]*Schema{
			"self": {
				Reference: &selfRef,
			},
		},
		AdditionalProperties: false,
		Required:             []string{},
	}

	s, err := FromType(reflect.TypeOf(test_types.OuterSelf{}), nil)
	assert.NoError(t, err)
	assert.Equal(t, expected, s)
}

func TestFromTypeError(t *testing.T) {
	type mapOfInts map[int]int
	_, err := FromType(reflect.TypeOf(mapOfInts{}), nil)
	assert.EqualError(t, err, "found map with non-string key: int")
}
