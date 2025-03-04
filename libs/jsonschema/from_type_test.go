package jsonschema

import (
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/jsonschema/test_types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromTypeBasic(t *testing.T) {
	type myStruct struct {
		S             string `json:"s"`
		I             *int   `json:"i,omitempty"`
		V             any    `json:"v,omitempty"`
		TriplePointer ***int `json:"triple_pointer,omitempty"`

		// These fields should be ignored in the resulting schema.
		NotAnnotated   string
		DashedTag      string `json:"-"`
		InternalTagged string `json:"internal_tagged" bundle:"internal"`
		ReadOnlyTagged string `json:"readonly_tagged" bundle:"readonly"`
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
			typ:  reflect.TypeOf(string("")),
			expected: Schema{
				Type: "string",
			},
		},
		{
			name: "bool",
			typ:  reflect.TypeOf(bool(true)),
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
					"interface": Schema{},
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
					"triple_pointer": {
						Reference: &intRef,
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
		})
	}
}

func TestGetStructFields(t *testing.T) {
	type InnerEmbeddedStruct struct {
		InnerField float64
	}

	type EmbeddedStructOne struct {
		FieldOne int

		*InnerEmbeddedStruct
	}

	type EmbeddedStructTwo struct {
		FieldTwo bool
	}

	type MyStruct struct {
		*EmbeddedStructOne
		EmbeddedStructTwo

		OuterField string
	}

	fields := getStructFields(reflect.TypeOf(MyStruct{}))
	assert.Len(t, fields, 4)
	assert.Equal(t, "OuterField", fields[0].Name)
	assert.Equal(t, "FieldOne", fields[1].Name)

	// InnerField occurring after FieldTwo ensures BFS as opposed to DFS traversal.
	assert.Equal(t, "FieldTwo", fields[2].Name)
	assert.Equal(t, "InnerField", fields[3].Name)
}

func TestHigherLevelEmbeddedFieldIsInSchema(t *testing.T) {
	type Inner struct {
		Override string `json:"override,omitempty"`
	}

	type EmbeddedOne struct {
		Inner
	}

	type EmbeddedTwo struct {
		Override int `json:"override,omitempty"`
	}

	type Outer struct {
		EmbeddedOne
		EmbeddedTwo
	}

	intRef := "#/$defs/int"
	expected := Schema{
		Type: "object",
		Definitions: map[string]any{
			"int": Schema{
				Type: "integer",
			},
		},
		Properties: map[string]*Schema{
			"override": {
				Reference: &intRef,
			},
		},
		AdditionalProperties: false,
		Required:             []string{},
	}

	s, err := FromType(reflect.TypeOf(Outer{}), nil)
	require.NoError(t, err)
	assert.Equal(t, expected, s)
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
			typ:  reflect.TypeOf(map[string]*Inner{}),
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
	// Maps with non-string keys should panic.
	type mapOfInts map[int]int
	assert.PanicsWithValue(t, "found map with non-string key: int", func() {
		_, err := FromType(reflect.TypeOf(mapOfInts{}), nil)
		require.NoError(t, err)
	})

	// Unsupported types should return an error.
	_, err := FromType(reflect.TypeOf(complex64(0)), nil)
	assert.EqualError(t, err, "unsupported type: complex64")
}

func TestFromTypeFunctionsArg(t *testing.T) {
	type myStruct struct {
		S string `json:"s"`
	}

	strRef := "#/$defs/string"
	expected := Schema{
		Type: "object",
		Definitions: map[string]any{
			"string": Schema{
				Type:        "string",
				Description: "a string",
				Enum:        []any{"a", "b", "c"},
			},
		},
		Properties: map[string]*Schema{
			"s": {
				Reference: &strRef,
			},
		},
		AdditionalProperties: false,
		Required:             []string{"s"},
	}

	addDescription := func(typ reflect.Type, s Schema) Schema {
		if typ.Kind() != reflect.String {
			return s
		}
		s.Description = "a string"
		return s
	}

	addEnums := func(typ reflect.Type, s Schema) Schema {
		if typ.Kind() != reflect.String {
			return s
		}
		s.Enum = []any{"a", "b", "c"}
		return s
	}

	s, err := FromType(reflect.TypeOf(myStruct{}), []func(reflect.Type, Schema) Schema{
		addDescription,
		addEnums,
	})
	assert.NoError(t, err)
	assert.Equal(t, expected, s)
}

func TestTypePath(t *testing.T) {
	type myStruct struct{}

	tcases := []struct {
		typ  reflect.Type
		path string
	}{
		{
			typ:  reflect.TypeOf(""),
			path: "string",
		},
		{
			typ:  reflect.TypeOf(int(0)),
			path: "int",
		},
		{
			typ:  reflect.TypeOf(true),
			path: "bool",
		},
		{
			typ:  reflect.TypeOf(float64(0)),
			path: "float64",
		},
		{
			typ:  reflect.TypeOf(myStruct{}),
			path: "github.com/databricks/cli/libs/jsonschema.myStruct",
		},
		{
			typ:  reflect.TypeOf([]int{}),
			path: "slice/int",
		},
		{
			typ:  reflect.TypeOf(map[string]int{}),
			path: "map/int",
		},
		{
			typ:  reflect.TypeOf([]myStruct{}),
			path: "slice/github.com/databricks/cli/libs/jsonschema.myStruct",
		},
		{
			typ:  reflect.TypeOf([][]map[string]map[string]myStruct{}),
			path: "slice/slice/map/map/github.com/databricks/cli/libs/jsonschema.myStruct",
		},
		{
			typ:  reflect.TypeOf(map[string]myStruct{}),
			path: "map/github.com/databricks/cli/libs/jsonschema.myStruct",
		},
	}

	for _, tc := range tcases {
		t.Run(tc.typ.String(), func(t *testing.T) {
			assert.Equal(t, tc.path, typePath(tc.typ))
		})
	}

	// Maps with non-string keys should panic.
	assert.PanicsWithValue(t, "found map with non-string key: int", func() {
		typePath(reflect.TypeOf(map[int]int{}))
	})
}
