package jsonschema

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromTypeBasic(t *testing.T) {
	type myStruct struct {
		S string `json:"s"`
		I int    `json:"i"`
	}

	strRef := "#/$defs/string"
	boolRef := "#/$defs/bool"
	intRef := "#/$defs/int"

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
				},
				AdditionalProperties: false,
				Required:             []string{"s", "i"},
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
