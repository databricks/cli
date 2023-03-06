package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSchemaForObject(t *testing.T) {
	s := func(s string) *string { return &s }
	spec := &openapi{
		Components: &Components{
			Schemas: map[string]*Schema{
				"fruits": {
					Type:        "object",
					Description: "fruits that are cool",
					Properties: map[string]*Schema{
						"mango": {
							Type:        "object",
							Reference:   s("#/components/schemas/mango"),
							Description: "a mango for my schema",
						},
						"guava": {
							Type:        "string",
							Description: "a guava for my schema",
						},
					},
				},
				"mango": {
					Type: "object",
					Properties: map[string]*Schema{
						"foo": {Reference: s("#/components/schemas/foo")},
					},
				},
				"foo": {Type: "number"},
			},
		},
	}
	fruitsSchema, err := spec.readResolvedSchema("#/components/schemas/fruits")
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
	s := func(s string) *string { return &s }
	spec := &openapi{
		Components: &Components{
			Schemas: map[string]*Schema{
				"fruits": {
					Type:        "object",
					Description: "fruits that are cool",
					Items: &Schema{
						Description: "some papayas, because papayas are fruits too",
						Reference:   s("#/components/schemas/papaya"),
					},
				},
				"papaya": {Type: "number"},
			},
		},
	}

	fruitsSchema, err := spec.readResolvedSchema("#/components/schemas/fruits")
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
	s := func(s string) *string { return &s }
	spec := &openapi{
		Components: &Components{
			Schemas: map[string]*Schema{
				"fruits": {
					Type:        "object",
					Description: "fruits that are meh",
					AdditionalProperties: &Schema{
						Type:        "number",
						Description: "watermelons. watermelons.",
						Reference:   s("#/components/schemas/watermelon"),
					},
				},
				"watermelon": {Type: "number"},
			},
		},
	}

	fruitsSchema, err := spec.readResolvedSchema("#/components/schemas/fruits")
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
	s := func(s string) *string { return &s }
	spec := &openapi{
		Components: &Components{
			Schemas: map[string]*Schema{
				"fruits": {
					Type:        "object",
					Description: "foo fighters fighting fruits",
					Reference:   s("#/components/schemas/foo"),
				},
				"foo": {
					Type:        "object",
					Description: "this description is ignored",
					Properties: map[string]*Schema{
						"abc": {
							Type: String,
						},
					},
				},
			},
		},
	}

	schema, err := spec.readResolvedSchema("#/components/schemas/fruits")
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
	s := func(s string) *string { return &s }
	spec := &openapi{
		Components: &Components{
			Schemas: map[string]*Schema{
				"fruits": {
					Type:        "object",
					Description: "foo fighters fighting fruits",
					Reference:   s("#/components/schemas/foo"),
				},
				"foo": {
					Type:        "object",
					Description: "this description is ignored",
					Properties: map[string]*Schema{
						"bar": {
							Type:      "object",
							Reference: s("#/components/schemas/foo"),
						},
					},
				},
			},
		},
	}

	_, err := spec.readResolvedSchema("#/components/schemas/fruits")
	assert.ErrorContains(t, err, "references loop detected. schema ref trace: #/components/schemas/fruits -> #/components/schemas/foo")
}

func TestCrossReferenceLoopErrors(t *testing.T) {
	s := func(s string) *string { return &s }
	spec := &openapi{
		Components: &Components{
			Schemas: map[string]*Schema{
				"fruits": {
					Type:        "object",
					Description: "foo fighters fighting fruits",
					Reference:   s("#/components/schemas/foo"),
				},
				"foo": {
					Type:        "object",
					Description: "this description is ignored",
					Properties: map[string]*Schema{
						"bar": {
							Type:      "object",
							Reference: s("#/components/schemas/fruits"),
						},
					},
				},
			},
		},
	}

	_, err := spec.readResolvedSchema("#/components/schemas/fruits")
	assert.ErrorContains(t, err, "references loop detected. schema ref trace: #/components/schemas/fruits -> #/components/schemas/foo")
}
