package schema

import (
	"container/list"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntSchema(t *testing.T) {
	var elemInt int

	expected :=
		`{
			"type": "number"
		}`

	Int, err := NewSchema(reflect.TypeOf(elemInt), nil)
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

	Int, err := NewSchema(reflect.TypeOf(elem), nil)
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

	Int, err := NewSchema(reflect.TypeOf(elem), nil)
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
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
			"additionalProperties": false,
			"required": [
				"int_val",
				"int8_val",
				"int16_val",
				"int32_val",
				"int64_val",
				"uint_val",
				"uint8_val",
				"uint16_val",
				"uint32_val",
				"uint64_val",
				"float32_val",
				"float64_val",
				"string_val",
				"bool_val"
			]
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
							"additionalProperties": false,
							"required": [
								"a",
								"b"
							]
						}
					},
					"additionalProperties": false,
					"required": [
						"bar"
					]
				}
			},
			"additionalProperties": false,
			"required": [
				"foo"
			]
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
					"additionalProperties": false,
					"required": [
						"my_map"
					]
				}
			},
			"additionalProperties": false,
			"required": [
				"bar"
			]
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
					"additionalProperties": false,
					"required": [
						"my_slice"
					]
				}
			},
			"additionalProperties": false,
			"required": [
				"bar"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfPrimitivesSchema(t *testing.T) {
	var elem map[string]int

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
				},
				"additionalProperties": false,
				"required": [
					"my_int"
				]
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestMapOfMapSchema(t *testing.T) {
	var elem map[string]map[string]int

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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

func TestSliceOfPrimitivesSchema(t *testing.T) {
	var elem []float32

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "array",
			"items": {
				"type": "number"
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestSliceOfSliceSchema(t *testing.T) {
	var elem [][]string

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "array",
			"items": {
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

func TestSliceOfMapSchema(t *testing.T) {
	var elem []map[string]int

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "array",
			"items": {
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

func TestSliceOfStructSchema(t *testing.T) {
	type Foo struct {
		MyInt int `json:"my_int"`
	}

	var elem []Foo

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "array",
			"items": {
				"type": "object",
				"properties": {
					"my_int": {
						"type": "number"
					}
				},
				"additionalProperties": false,
				"required": [
					"my_int"
				]
			}
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestEmbeddedStructSchema(t *testing.T) {
	type Location struct {
		Country string `json:"country"`
		State   string `json:"state,omitempty"`
	}

	type Person struct {
		Name string   `json:"name"`
		Age  int      `json:"age,omitempty"`
		Home Location `json:"home"`
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
				"home": {
					"type": "object",
					"properties": {
						"country": {
							"type": "string"
						},
						"state": {
							"type": "string"
						}
					},
					"additionalProperties": false,
					"required": [
						"country"
					]
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
									"home": {
										"type": "object",
										"properties": {
											"country": {
												"type": "string"
											},
											"state": {
												"type": "string"
											}
										},
										"additionalProperties": false,
										"required": [
											"country"
										]
									},
									"name": {
										"type": "string"
									}
								},
								"additionalProperties": false,
								"required": [
									"name",
									"home"
								]
							}
						}
					},
					"additionalProperties": false,
					"required": [
						"events"
					]
				},
				"state": {
					"type": "string"
				}
			},
			"additionalProperties": false,
			"required": [
				"plot",
				"name",
				"home",
				"country"
			]
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
			},
			"additionalProperties": false,
			"required": [
				"bar"
			]
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
			},
			"additionalProperties": false,
			"required": [
				"bar"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestPointerInStructSchema(t *testing.T) {

	type Bar struct {
		PtrVal2 *int `json:"ptr_val2"`
	}

	type Foo struct {
		PtrInt    *int    `json:"ptr_int"`
		PtrString *string `json:"ptr_string"`
		FloatVal  float32 `json:"float_val"`
		PtrBar    *Bar    `json:"ptr_bar"`
		Bar       *Bar    `json:"bar"`
	}

	elem := Foo{}

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"properties": {
				"bar": {
					"type": "object",
					"properties": {
						"ptr_val2": {
							"type": "number"
						}
					},
					"additionalProperties": false,
					"required": [
						"ptr_val2"
					]
				},
				"float_val": {
					"type": "number"
				},
				"ptr_bar": {
					"type": "object",
					"properties": {
						"ptr_val2": {
							"type": "number"
						}
					},
					"additionalProperties": false,
					"required": [
						"ptr_val2"
					]
				},
				"ptr_int": {
					"type": "number"
				},
				"ptr_string": {
					"type": "string"
				}
			},
			"additionalProperties": false,
			"required": [
				"ptr_int",
				"ptr_string",
				"float_val",
				"ptr_bar",
				"bar"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

// TODO: last test to do once all the todos are done
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

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
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
					},
					"additionalProperties": false,
					"required": [
						"name"
					]
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
					},
					"additionalProperties": false,
					"required": [
						"stakes"
					]
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
					},
					"additionalProperties": false,
					"required": [
						"name"
					]
				}
			},
			"additionalProperties": false,
			"required": [
				"hero",
				"villian",
				"plot"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestFieldsWithoutOmitEmptyAreRequired(t *testing.T) {

	type Papaya struct {
		A int    `json:"a,string,omitempty"`
		B string `json:"b"`
	}

	type MyStruct struct {
		Foo    string  `json:"-,omitempty"`
		Bar    int     `json:"bar"`
		Apple  int     `json:"apple,omitempty"`
		Mango  int     `json:",omitempty"`
		Guava  int     `json:","`
		Papaya *Papaya `json:"papaya,"`
	}

	elem := MyStruct{}

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"properties": {
				"apple": {
					"type": "number"
				},
				"bar": {
					"type": "number"
				},
				"papaya": {
					"type": "object",
					"properties": {
						"a": {
							"type": "number"
						},
						"b": {
							"type": "string"
						}
					},
					"additionalProperties": false,
					"required": [
						"b"
					]
				}
			},
			"additionalProperties": false,
			"required": [
				"bar",
				"papaya"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestDocIngestionInSchema(t *testing.T) {
	docs := &Docs{
		Documentation: "docs for root",
		Children: map[string]Docs{
			"my_struct": {
				Documentation: "docs for my struct",
			},
			"my_val": {
				Documentation: "docs for my val",
			},
			"my_slice": {
				Documentation: "docs for my slice",
				Children: map[string]Docs{
					"guava": {
						Documentation: "docs for guava",
					},
					"pineapple": {
						Documentation: "docs for pineapple",
					},
				},
			},
			"my_map": {
				Documentation: "docs for my map",
				Children: map[string]Docs{
					"apple": {
						Documentation: "docs for apple",
					},
					"mango": {
						Documentation: "docs for mango",
					},
				},
			},
		},
	}

	type Foo struct {
		Apple int `json:"apple"`
		Mango int `json:"mango"`
	}

	type Bar struct {
		Guava     int `json:"guava"`
		Pineapple int `json:"pineapple"`
	}

	type MyStruct struct {
		A string `json:"a"`
	}

	type Root struct {
		MyStruct *MyStruct       `json:"my_struct"`
		MyVal    int             `json:"my_val"`
		MySlice  []Bar           `json:"my_slice"`
		MyMap    map[string]*Foo `json:"my_map"`
	}

	elem := Root{}

	schema, err := NewSchema(reflect.TypeOf(elem), docs)
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"description": "docs for root",
			"properties": {
				"my_map": {
					"type": "object",
					"description": "docs for my map",
					"additionalProperties": {
						"type": "object",
						"description": "docs for my map",
						"properties": {
							"apple": {
								"type": "number",
								"description": "docs for apple"
							},
							"mango": {
								"type": "number",
								"description": "docs for mango"
							}
						},
						"additionalProperties": false,
						"required": [
							"apple",
							"mango"
						]
					}
				},
				"my_slice": {
					"type": "array",
					"description": "docs for my slice",
					"items": {
						"type": "object",
						"properties": {
							"guava": {
								"type": "number",
								"description": "docs for guava"
							},
							"pineapple": {
								"type": "number",
								"description": "docs for pineapple"
							}
						},
						"additionalProperties": false,
						"required": [
							"guava",
							"pineapple"
						]
					}
				},
				"my_struct": {
					"type": "object",
					"description": "docs for my struct",
					"properties": {
						"a": {
							"type": "string"
						}
					},
					"additionalProperties": false,
					"required": [
						"a"
					]
				},
				"my_val": {
					"type": "number",
					"description": "docs for my val"
				}
			},
			"additionalProperties": false,
			"required": [
				"my_struct",
				"my_val",
				"my_slice",
				"my_map"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestErrorOnMapWithoutStringKey(t *testing.T) {
	type Foo struct {
		Bar map[int]string `json:"bar"`
	}
	elem := Foo{}
	_, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.ErrorContains(t, err, "only strings map keys are valid. key type: int")
}

func TestErrorIfStructRefersToItself(t *testing.T) {
	type Foo struct {
		MyFoo *Foo `json:"my_foo"`
	}

	elem := Foo{}
	_, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.ErrorContains(t, err, "ERROR] cycle detected. traversal trace: root -> my_foo")
}

func TestErrorIfStructHasLoop(t *testing.T) {
	type Apple struct {
		MyVal   int `json:"my_val"`
		MyMango struct {
			MyGuava struct {
				MyPapaya struct {
					MyApple *Apple `json:"my_apple"`
				} `json:"my_papaya"`
			} `json:"my_guava"`
		} `json:"my_mango"`
	}

	elem := Apple{}
	_, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.ErrorContains(t, err, "[ERROR] cycle detected. traversal trace: root -> my_mango -> my_guava -> my_papaya -> my_apple")
}

func TestInterfaceGeneratesEmptySchema(t *testing.T) {
	type Foo struct {
		Apple int         `json:"apple"`
		Mango interface{} `json:"mango"`
	}

	elem := Foo{}

	schema, err := NewSchema(reflect.TypeOf(elem), nil)
	assert.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expected :=
		`{
			"type": "object",
			"properties": {
				"apple": {
					"type": "number"
				},
				"mango": {}
			},
			"additionalProperties": false,
			"required": [
				"apple",
				"mango"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

// A toy test to generate the schema for bundle. Will be removed once we have a
// command to generate the json schema
// func TestBundleSchema(t *testing.T) {
// 	elem := config.Root{}

// 	docs, err := LoadDocs("./bundle_config_docs.yml")
// 	require.NoError(t, err)

// 	schema, err := NewSchema(reflect.TypeOf(elem), docs)
// 	assert.NoError(t, err)

// 	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
// 	assert.NoError(t, err)

// 	t.Log("[DEBUG] actual: ", string(jsonSchema))
// 	assert.True(t, false)
// }
