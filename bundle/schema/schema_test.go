package schema

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: add tests to assert that these are valid json schemas. Maybe validate some
// json/yaml documents againts them, by unmarshalling a value

func TestNumberStringBooleanSchema(t *testing.T) {
	type Foo struct {
		IntVal   int   `json:"int_val"`
		Int8Val  int8  `json:"int8_val"`
		Int16Val int16 `json:"int16_val"`
		Int32Val int32 `json:"int32_val"`
		Int64Val int64 `json:"int64_val"`

		Uint8Val  int8  `json:"uint8_val"`
		Uint16Val int16 `json:"uint16_val"`
		Uint32Val int32 `json:"uint32_val"`
		Uint64Val int64 `json:"uint64_val"`

		Float32Val int64 `json:"float32_val"`
		Float64Val int64 `json:"float64_val"`

		StringVal string `json:"string_val"`

		BoolVal string `json:"bool_val"`
	}

	elem := Foo{}

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"properties": {
				"bool_val": {
					"type": "string"
				},
				"float32_val": {
					"type": "number"
				},
				"float64_val": {
					"type": "number"
				},
				"int16_val": {
					"type": "number"
				},
				"int32_val": {
					"type": "number"
				},
				"int64_val": {
					"type": "number"
				},
				"int8_val": {
					"type": "number"
				},
				"int_val": {
					"type": "number"
				},
				"string_val": {
					"type": "string"
				},
				"uint16_val": {
					"type": "number"
				},
				"uint32_val": {
					"type": "number"
				},
				"uint64_val": {
					"type": "number"
				},
				"uint8_val": {
					"type": "number"
				}
			}
		}`

	fmt.Println("[DEBUG] actual: ", string(jsonSchema))
	fmt.Println("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestObjectSchema(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}

	type Plot struct {
		Stakes []string `json:"stakes"`
	}

	type Story struct {
		Hero    Person `json:"hero"`
		Villian Person `json:"villian"`
		Plot    Plot   `json:"plot"`
	}

	elem := Story{}

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"properties": {
				"hero": {
					"type": "object",
					"properties": {
						"age": {
							"type": "number"
						},
						"name": {
							"type": "string"
						}
					}
				},
				"plot": {
					"type": "object",
					"properties": {
						"stakes": {
							"type": "array",
							"items": {
								"type": "string"
							}
						}
					}
				},
				"villian": {
					"type": "object",
					"properties": {
						"age": {
							"type": "number"
						},
						"name": {
							"type": "string"
						}
					}
				}
			}
		}`

	fmt.Println("[DEBUG] actual: ", string(jsonSchema))
	fmt.Println("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}
