package schema

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaToDocs(t *testing.T) {
	jsonSchema := &schema.Schema{
		Type:        "object",
		Description: "root doc",
		Properties: map[string]*schema.Schema{
			"foo": {Type: "number", Description: "foo doc"},
			"bar": {Type: "string"},
			"octave": {
				Type:                 "object",
				AdditionalProperties: &schema.Schema{Type: "number"},
				Description:          "octave docs",
			},
			"scales": {
				Type:        "object",
				Description: "scale docs",
				Items:       &schema.Schema{Type: "string"},
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
