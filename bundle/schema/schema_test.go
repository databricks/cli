package schema

import (
	"container/list"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: add tests to assert that these are valid json schemas. Maybe validate some
// json/yaml documents againts them, by unmarshalling a value

// TODO: See that all golang reflect types are covered (reasonalble limits) within
// these tests

// TODO: test what json schema is generated for an interface{} type. Make sure the behavior makes sense

// TODO: Have a test that combines multiple different cases
// TODO: have a test for what happens when omitempty in different cases: primitives, object, map, array

func TestIntSchema(t *testing.T) {
	var elemInt int

	expected :=
		`{
			"type": "number"
		}`

	Int, err := NewSchema(reflect.TypeOf(elemInt))
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(Int, "		", "	")
	assert.NoError(t, err)

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestBooleanSchema(t *testing.T) {
	var elem bool

	expected :=
		`{
			"type": "boolean"
		}`

	Int, err := NewSchema(reflect.TypeOf(elem))
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(Int, "		", "	")
	assert.NoError(t, err)

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestStringSchema(t *testing.T) {
	var elem string

	expected :=
		`{
			"type": "string"
		}`

	Int, err := NewSchema(reflect.TypeOf(elem))
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(Int, "		", "	")
	assert.NoError(t, err)

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestStructOfPrimitivesSchema(t *testing.T) {
	type Foo struct {
		IntVal   int   `json:"int_val"`
		Int8Val  int8  `json:"int8_val"`
		Int16Val int16 `json:"int16_val"`
		Int32Val int32 `json:"int32_val"`
		Int64Val int64 `json:"int64_val"`

		UIntVal   uint   `json:"uint_val"`
		Uint8Val  uint8  `json:"uint8_val"`
		Uint16Val uint16 `json:"uint16_val"`
		Uint32Val uint32 `json:"uint32_val"`
		Uint64Val uint64 `json:"uint64_val"`

		Float32Val float32 `json:"float32_val"`
		Float64Val float64 `json:"float64_val"`

		StringVal string `json:"string_val"`

		BoolVal bool `json:"bool_val"`
	}

	elem := Foo{}

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		` {
			"type": "object",
			"properties": {
				"bool_val": {
					"type": "boolean"
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
				},
				"uint_val": {
					"type": "number"
				}
			},
			"additionalProperties": false
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestStructOfStructsSchema(t *testing.T) {
	type Bar struct {
		A int    `json:"a"`
		B string `json:"b,string"`
	}

	type Foo struct {
		Bar Bar `json:"bar"`
	}

	type MyStruct struct {
		Foo Foo `json:"foo"`
	}

	elem := MyStruct{}

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"properties": {
				"foo": {
					"type": "object",
					"properties": {
						"bar": {
							"type": "object",
							"properties": {
								"a": {
									"type": "number"
								},
								"b": {
									"type": "string"
								}
							},
							"additionalProperties": false
						}
					},
					"additionalProperties": false
				}
			},
			"additionalProperties": false
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestStructOfMapsSchema(t *testing.T) {
	type Bar struct {
		MyMap map[string]int `json:"my_map"`
	}

	type Foo struct {
		Bar Bar `json:"bar"`
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
				"bar": {
					"type": "object",
					"properties": {
						"my_map": {
							"type": "object",
							"additionalProperties": {
								"type": "number"
							}
						}
					},
					"additionalProperties": false
				}
			},
			"additionalProperties": false
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestStructOfSliceSchema(t *testing.T) {
	type Bar struct {
		MySlice []string `json:"my_slice"`
	}

	type Foo struct {
		Bar Bar `json:"bar"`
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
				"bar": {
					"type": "object",
					"properties": {
						"my_slice": {
							"type": "array",
							"items": {
								"type": "string"
							}
						}
					},
					"additionalProperties": false
				}
			},
			"additionalProperties": false
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfPrimitivesSchema(t *testing.T) {
	var elem map[string]int

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"additionalProperties": {
				"type": "number"
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfStructSchema(t *testing.T) {
	type Foo struct {
		MyInt int `json:"my_int"`
	}

	var elem map[string]Foo

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"additionalProperties": {
				"type": "object",
				"properties": {
					"my_int": {
						"type": "number"
					}
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfMapSchema(t *testing.T) {
	var elem map[string]map[string]int

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"additionalProperties": {
				"type": "object",
				"additionalProperties": {
					"type": "number"
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfSliceSchema(t *testing.T) {
	var elem map[string][]string

	schema, err := NewSchema(reflect.TypeOf(elem))
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"additionalProperties": {
				"type": "array",
				"items": {
					"type": "string"
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

func TestErrorWithTrace(t *testing.T) {
	debugTrace := list.New()
	err := errWithTrace("with empty trace", debugTrace)
	assert.ErrorContains(t, err, "[ERROR] with empty trace. traversal trace: root")

	debugTrace.PushBack("resources")
	err = errWithTrace("with depth = 1", debugTrace)
	assert.ErrorContains(t, err, "[ERROR] with depth = 1. traversal trace: root -> resources")

	debugTrace.PushBack("pipelines")
	debugTrace.PushBack("datasets")
	err = errWithTrace("with depth = 4", debugTrace)
	assert.ErrorContains(t, err, "[ERROR] with depth = 4. traversal trace: root -> resources -> pipelines -> datasets")
}

func TestNonAnnotatedFieldsAreSkipped(t *testing.T) {
	type MyStruct struct {
		Foo string
		Bar int `json:"bar"`
	}

	elem := MyStruct{}

	schema, err := NewSchema(reflect.TypeOf(elem))
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"properties": {
				"bar": {
					"type": "number"
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestDashFieldsAreSkipped(t *testing.T) {
	type MyStruct struct {
		Foo string `json:"-"`
		Bar int    `json:"bar"`
	}

	elem := MyStruct{}

	schema, err := NewSchema(reflect.TypeOf(elem))
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"properties": {
				"bar": {
					"type": "number"
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
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
