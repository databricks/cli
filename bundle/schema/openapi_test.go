package schema

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSchemaForObject(t *testing.T) {
	specString := `
	{
		"components": {
			"schemas": {
				"foo": {
					"type": "number"
				},
				"fruits": {
					"type": "object",
					"description": "fruits that are cool",
					"properties": {
						"guava": {
							"type": "string",
							"description": "a guava for my schema"
						},
						"mango": {
							"type": "object",
							"description": "a mango for my schema",
							"$ref": "#/components/schemas/mango"
						}
					}
				},
				"mango": {
					"type": "object",
					"properties": {
						"foo": {
							"$ref": "#/components/schemas/foo"
						}
					}
				}
			}
		}
	}
	`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	fruitsSchema, err := reader.readResolvedSchema("#/components/schemas/fruits")
	require.NoError(t, err)

	fruitsSchemaJson, err := json.MarshalIndent(fruitsSchema, "		", "	")
	require.NoError(t, err)

	expected := `{
			"type": "object",
			"description": "fruits that are cool",
			"properties": {
				"guava": {
					"type": "string",
					"description": "a guava for my schema"
				},
				"mango": {
					"type": "object",
					"description": "a mango for my schema",
					"properties": {
						"foo": {
							"type": "number"
						}
					}
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(fruitsSchemaJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(fruitsSchemaJson))
}

func TestReadSchemaForArray(t *testing.T) {
	specString := `
	{
		"components": {
			"schemas": {
				"fruits": {
					"type": "object",
					"description": "fruits that are cool",
					"items": {
						"description": "some papayas, because papayas are fruits too",
						"$ref": "#/components/schemas/papaya"
					}
				},
				"papaya": {
					"type": "number"
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	fruitsSchema, err := reader.readResolvedSchema("#/components/schemas/fruits")
	require.NoError(t, err)

	fruitsSchemaJson, err := json.MarshalIndent(fruitsSchema, "		", "	")
	require.NoError(t, err)

	expected := `{
			"type": "object",
			"description": "fruits that are cool",
			"items": {
				"type": "number",
				"description": "some papayas, because papayas are fruits too"
			}
		}`

	t.Log("[DEBUG] actual: ", string(fruitsSchemaJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(fruitsSchemaJson))
}

func TestReadSchemaForMap(t *testing.T) {
	specString := `{
		"components": {
			"schemas": {
				"fruits": {
					"type": "object",
					"description": "fruits that are meh",
					"additionalProperties": {
						"description": "watermelons. watermelons.",
						"$ref": "#/components/schemas/watermelon"
					}
				},
				"watermelon": {
					"type": "number"
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	fruitsSchema, err := reader.readResolvedSchema("#/components/schemas/fruits")
	require.NoError(t, err)

	fruitsSchemaJson, err := json.MarshalIndent(fruitsSchema, "		", "	")
	require.NoError(t, err)

	expected := `{
			"type": "object",
			"description": "fruits that are meh",
			"additionalProperties": {
				"type": "number",
				"description": "watermelons. watermelons."
			}
		}`

	t.Log("[DEBUG] actual: ", string(fruitsSchemaJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(fruitsSchemaJson))
}

func TestRootReferenceIsResolved(t *testing.T) {
	specString := `{
		"components": {
			"schemas": {
				"foo": {
					"type": "object",
					"description": "this description is ignored",
					"properties": {
						"abc": {
							"type": "string"
						}
					}
				},
				"fruits": {
					"type": "object",
					"description": "foo fighters fighting fruits",
					"$ref": "#/components/schemas/foo"
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	schema, err := reader.readResolvedSchema("#/components/schemas/fruits")
	require.NoError(t, err)
	fruitsSchemaJson, err := json.MarshalIndent(schema, "		", "	")
	require.NoError(t, err)

	expected := `{
			"type": "object",
			"description": "foo fighters fighting fruits",
			"properties": {
				"abc": {
					"type": "string"
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(fruitsSchemaJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(fruitsSchemaJson))
}

func TestSelfReferenceLoopErrors(t *testing.T) {
	specString := `{
		"components": {
			"schemas": {
				"foo": {
					"type": "object",
					"description": "this description is ignored",
					"properties": {
						"bar": {
							"type": "object",
							"$ref": "#/components/schemas/foo"
						}
					}
				},
				"fruits": {
					"type": "object",
					"description": "foo fighters fighting fruits",
					"$ref": "#/components/schemas/foo"
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	_, err = reader.readResolvedSchema("#/components/schemas/fruits")
	assert.ErrorContains(t, err, "references loop detected. traversal trace:  -> #/components/schemas/fruits -> #/components/schemas/foo")
}

func TestCrossReferenceLoopErrors(t *testing.T) {
	specString := `{
		"components": {
			"schemas": {
				"foo": {
					"type": "object",
					"description": "this description is ignored",
					"properties": {
						"bar": {
							"type": "object",
							"$ref": "#/components/schemas/fruits"
						}
					}
				},
				"fruits": {
					"type": "object",
					"description": "foo fighters fighting fruits",
					"$ref": "#/components/schemas/foo"
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	_, err = reader.readResolvedSchema("#/components/schemas/fruits")
	assert.ErrorContains(t, err, "references loop detected. traversal trace:  -> #/components/schemas/fruits -> #/components/schemas/foo")
}

func TestReferenceResolutionForMapInObject(t *testing.T) {
	specString := `
	{
		"components": {
			"schemas": {
				"foo": {
					"type": "number"
				},
				"fruits": {
					"type": "object",
					"description": "fruits that are cool",
					"properties": {
						"guava": {
							"type": "string",
							"description": "a guava for my schema"
						},
						"mangos": {
							"type": "object",
							"description": "multiple mangos",
							"$ref": "#/components/schemas/mango"
						}
					}
				},
				"mango": {
					"type": "object",
					"additionalProperties": {
						"description": "a single mango",
						"$ref": "#/components/schemas/foo"
					}
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	fruitsSchema, err := reader.readResolvedSchema("#/components/schemas/fruits")
	require.NoError(t, err)

	fruitsSchemaJson, err := json.MarshalIndent(fruitsSchema, "		", "	")
	require.NoError(t, err)

	expected := `{
			"type": "object",
			"description": "fruits that are cool",
			"properties": {
				"guava": {
					"type": "string",
					"description": "a guava for my schema"
				},
				"mangos": {
					"type": "object",
					"description": "multiple mangos",
					"additionalProperties": {
						"type": "number",
						"description": "a single mango"
					}
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(fruitsSchemaJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(fruitsSchemaJson))
}

func TestReferenceResolutionForArrayInObject(t *testing.T) {
	specString := `{
		"components": {
			"schemas": {
				"foo": {
					"type": "number"
				},
				"fruits": {
					"type": "object",
					"description": "fruits that are cool",
					"properties": {
						"guava": {
							"type": "string",
							"description": "a guava for my schema"
						},
						"mangos": {
							"type": "object",
							"description": "multiple mangos",
							"$ref": "#/components/schemas/mango"
						}
					}
				},
				"mango": {
					"type": "object",
					"items": {
						"description": "a single mango",
						"$ref": "#/components/schemas/foo"
					}
				}
			}
		}
	}`
	spec := &openapi.Specification{}
	reader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*Schema),
	}
	err := json.Unmarshal([]byte(specString), spec)
	require.NoError(t, err)

	fruitsSchema, err := reader.readResolvedSchema("#/components/schemas/fruits")
	require.NoError(t, err)

	fruitsSchemaJson, err := json.MarshalIndent(fruitsSchema, "		", "	")
	require.NoError(t, err)

	expected := `{
			"type": "object",
			"description": "fruits that are cool",
			"properties": {
				"guava": {
					"type": "string",
					"description": "a guava for my schema"
				},
				"mangos": {
					"type": "object",
					"description": "multiple mangos",
					"items": {
						"type": "number",
						"description": "a single mango"
					}
				}
			}
		}`

	t.Log("[DEBUG] actual: ", string(fruitsSchemaJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(fruitsSchemaJson))
}
