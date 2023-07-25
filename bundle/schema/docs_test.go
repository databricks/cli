package schema

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaToDocs(t *testing.T) {
	jsonSchema := &jsonschema.Schema{
		Type:        "object",
		Description: "root doc",
		Properties: map[string]*jsonschema.Schema{
			"foo": {Type: "number", Description: "foo doc"},
			"bar": {Type: "string"},
			"octave": {
				Type:                 "object",
				AdditionalProperties: &jsonschema.Schema{Type: "number"},
				Description:          "octave docs",
			},
			"scales": {
				Type:        "object",
				Description: "scale docs",
				Items:       &jsonschema.Schema{Type: "string"},
			},
		},
	}
	docs := schemaToDocs(jsonSchema)
	docsJson, err := json.MarshalIndent(docs, "		", "	")
	require.NoError(t, err)

	expected :=
		`{
			"description": "root doc",
			"properties": {
				"bar": {
					"description": ""
				},
				"foo": {
					"description": "foo doc"
				},
				"octave": {
					"description": "octave docs",
					"additionalproperties": {
						"description": ""
					}
				},
				"scales": {
					"description": "scale docs",
					"items": {
						"description": ""
					}
				}
			}
		}`
	t.Log("[DEBUG] actual: ", string(docsJson))
	t.Log("[DEBUG] expected: ", expected)
	assert.Equal(t, expected, string(docsJson))
}
