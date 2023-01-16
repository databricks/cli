package schema

import (
	"encoding/json"
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

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
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

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestSliceOfObjectsSchema(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}

	type Plot struct {
		MainCharacters []Person `json:"main_characters"`
	}

	type Story struct {
		Plot Plot `json:"plot"`
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
				"plot": {
					"type": "object",
					"properties": {
						"main_characters": {
							"type": "array",
							"items": {
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
					}
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfObjectsSchema(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}

	type Plot struct {
		Events map[string]Person `json:"events"`
	}

	type Story struct {
		Plot Plot `json:"plot"`
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
				"plot": {
					"type": "object",
					"properties": {
						"events": {
							"type": "object",
							"additionalProperties": {
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
					}
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestEmbeddedStructSchema(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}

	type Location struct {
		Country string `json:"country"`
		State   string `json:"state,omitempty"`
	}

	type Plot struct {
		Events map[string]Person `json:"events"`
	}

	type Story struct {
		Plot Plot `json:"plot"`
		*Person
		Location
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
				"age": {
					"type": "number"
				},
				"country": {
					"type": "string"
				},
				"name": {
					"type": "string"
				},
				"plot": {
					"type": "object",
					"properties": {
						"events": {
							"type": "object",
							"additionalProperties": {
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
					}
				},
				"state": {
					"type": "string"
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

// // Only for testing bundle, will be removed
// func TestBundleSchema(t *testing.T) {
// 	elem := config.Root{}

// 	schema, err := NewSchema(reflect.TypeOf(elem))
// 	assert.NoError(t, err)

// 	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
// 	assert.NoError(t, err)

// 	expected :=
// 		``

// 	t.Log("[DEBUG] actual: ", string(jsonSchema))
// 	t.Log("[DEBUG] expected: ", expected)
// 	assert.Equal(t, expected, string(jsonSchema))
// }
