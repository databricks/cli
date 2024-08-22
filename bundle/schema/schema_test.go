package schema

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Add a test that checks the primitive overrides for reference regexs work.
// Basically that the custom override for bundle regex works.


func TestGenericSchema(t *testing.T) {
	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}

	type Plot struct {
		Stakes  []string          `json:"stakes"`
		Deaths  []Person          `json:"deaths"`
		Murders map[string]Person `json:"murders"`
	}

	type Wedding struct {
		Hidden string `json:","`
		Groom  Person `json:"groom"`
		Bride  Person `json:"bride"`
		Plots  []Plot `json:"plots"`
	}

	type Story struct {
		Hero     *Person   `json:"hero"`
		Villian  Person    `json:"villian,omitempty"`
		Weddings []Wedding `json:"weddings"`
	}

	elem := Story{}

	schema, err := New(reflect.TypeOf(elem), nil)
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
							"anyOf": [
								{
									"type": "number"
								},
								{
									"type": "string",
									"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
								}
							]
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
				"villian": {
					"type": "object",
					"properties": {
						"age": {
							"anyOf": [
								{
									"type": "number"
								},
								{
									"type": "string",
									"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
								}
							]
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
				"weddings": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"bride": {
								"type": "object",
								"properties": {
									"age": {
										"anyOf": [
											{
												"type": "number"
											},
											{
												"type": "string",
												"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
											}
										]
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
							"groom": {
								"type": "object",
								"properties": {
									"age": {
										"anyOf": [
											{
												"type": "number"
											},
											{
												"type": "string",
												"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
											}
										]
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
							"plots": {
								"type": "array",
								"items": {
									"type": "object",
									"properties": {
										"deaths": {
											"type": "array",
											"items": {
												"type": "object",
												"properties": {
													"age": {
														"anyOf": [
															{
																"type": "number"
															},
															{
																"type": "string",
																"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
															}
														]
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
										"murders": {
											"type": "object",
											"additionalProperties": {
												"type": "object",
												"properties": {
													"age": {
														"anyOf": [
															{
																"type": "number"
															},
															{
																"type": "string",
																"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
															}
														]
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
										"stakes": {
											"type": "array",
											"items": {
												"type": "string"
											}
										}
									},
									"additionalProperties": false,
									"required": [
										"stakes",
										"deaths",
										"murders"
									]
								}
							}
						},
						"additionalProperties": false,
						"required": [
							"groom",
							"bride",
							"plots"
						]
					}
				}
			},
			"additionalProperties": false,
			"required": [
				"hero",
				"weddings"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(jsonSchema))
}

func TestDocIngestionForObject(t *testing.T) {
	docs := &Docs{
		Description: "docs for root",
		Properties: map[string]*Docs{
			"my_struct": {
				Description: "docs for my struct",
				Properties: map[string]*Docs{
					"a": {
						Description: "docs for a",
					},
					"c": {
						Description: "docs for c which does not exist on my_struct",
					},
				},
			},
		},
	}

	type MyStruct struct {
		A string `json:"a"`
		B int    `json:"b"`
	}

	type Root struct {
		MyStruct *MyStruct `json:"my_struct"`
	}

	elem := Root{}

	schema, err := New(reflect.TypeOf(elem), docs)
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"description": "docs for root",
			"properties": {
				"my_struct": {
					"type": "object",
					"description": "docs for my struct",
					"properties": {
						"a": {
							"type": "string",
							"description": "docs for a"
						},
						"b": {
							"anyOf": [
								{
									"type": "number"
								},
								{
									"type": "string",
									"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
								}
							]
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
				"my_struct"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestDocIngestionForSlice(t *testing.T) {
	docs := &Docs{
		Description: "docs for root",
		Properties: map[string]*Docs{
			"my_slice": {
				Description: "docs for my slice",
				Items: &Docs{
					Properties: map[string]*Docs{
						"guava": {
							Description: "docs for guava",
						},
						"pineapple": {
							Description: "docs for pineapple",
						},
						"watermelon": {
							Description: "docs for watermelon which does not exist in schema",
						},
					},
				},
			},
		},
	}

	type Bar struct {
		Guava     int `json:"guava"`
		Pineapple int `json:"pineapple"`
	}

	type Root struct {
		MySlice []Bar `json:"my_slice"`
	}

	elem := Root{}

	schema, err := New(reflect.TypeOf(elem), docs)
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"description": "docs for root",
			"properties": {
				"my_slice": {
					"type": "array",
					"description": "docs for my slice",
					"items": {
						"type": "object",
						"properties": {
							"guava": {
								"description": "docs for guava",
								"anyOf": [
									{
										"type": "number"
									},
									{
										"type": "string",
										"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
									}
								]
							},
							"pineapple": {
								"description": "docs for pineapple",
								"anyOf": [
									{
										"type": "number"
									},
									{
										"type": "string",
										"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
									}
								]
							}
						},
						"additionalProperties": false,
						"required": [
							"guava",
							"pineapple"
						]
					}
				}
			},
			"additionalProperties": false,
			"required": [
				"my_slice"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestDocIngestionForMap(t *testing.T) {
	docs := &Docs{
		Description: "docs for root",
		Properties: map[string]*Docs{
			"my_map": {
				Description: "docs for my map",
				AdditionalProperties: &Docs{
					Properties: map[string]*Docs{
						"apple": {
							Description: "docs for apple",
						},
						"mango": {
							Description: "docs for mango",
						},
						"watermelon": {
							Description: "docs for watermelon which does not exist in schema",
						},
						"papaya": {
							Description: "docs for papaya which does not exist in schema",
						},
					},
				},
			},
		},
	}

	type Foo struct {
		Apple int `json:"apple"`
		Mango int `json:"mango"`
	}

	type Root struct {
		MyMap map[string]*Foo `json:"my_map"`
	}

	elem := Root{}

	schema, err := New(reflect.TypeOf(elem), docs)
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
						"properties": {
							"apple": {
								"description": "docs for apple",
								"anyOf": [
									{
										"type": "number"
									},
									{
										"type": "string",
										"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
									}
								]
							},
							"mango": {
								"description": "docs for mango",
								"anyOf": [
									{
										"type": "number"
									},
									{
										"type": "string",
										"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
									}
								]
							}
						},
						"additionalProperties": false,
						"required": [
							"apple",
							"mango"
						]
					}
				}
			},
			"additionalProperties": false,
			"required": [
				"my_map"
			]
		}`

	t.Log("[DEBUG] actual: ", string(jsonSchema))
	t.Log("[DEBUG] expected: ", expectedSchema)

	assert.Equal(t, expectedSchema, string(jsonSchema))
}

func TestDocIngestionForTopLevelPrimitive(t *testing.T) {
	docs := &Docs{
		Description: "docs for root",
		Properties: map[string]*Docs{
			"my_val": {
				Description: "docs for my val",
			},
		},
	}

	type Root struct {
		MyVal int `json:"my_val"`
	}

	elem := Root{}

	schema, err := New(reflect.TypeOf(elem), docs)
	require.NoError(t, err)

	jsonSchema, err := json.MarshalIndent(schema, "		", "	")
	assert.NoError(t, err)

	expectedSchema :=
		`{
			"type": "object",
			"description": "docs for root",
			"properties": {
				"my_val": {
					"description": "docs for my val",
					"anyOf": [
						{
							"type": "number"
						},
						{
							"type": "string",
							"pattern": "\\$\\{([a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\\[[0-9]+\\])*)*(\\[[0-9]+\\])*)\\}"
						}
					]
				}
			},
			"additionalProperties": false,
			"required": [
				"my_val"
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
	_, err := New(reflect.TypeOf(elem), nil)
	assert.ErrorContains(t, err, "only strings map keys are valid. key type: int")
}

func TestErrorIfStructRefersToItself(t *testing.T) {
	type Foo struct {
		MyFoo *Foo `json:"my_foo"`
	}

	elem := Foo{}
	_, err := New(reflect.TypeOf(elem), nil)
	assert.ErrorContains(t, err, "cycle detected. traversal trace: root -> my_foo")
}
