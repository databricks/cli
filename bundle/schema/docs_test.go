package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaToDocs(t *testing.T) {
	schema := &Schema{
		Type:        "object",
		Description: "root doc",
		Properties: map[string]*Schema{
			"foo": {Type: "number", Description: "foo doc"},
			"bar": {Type: "string"},
			"octave": {
				Type:                 "object",
				AdditionalProperties: &Schema{Type: "number"},
				Description:          "octave docs",
			},
			"scales": {
				Type: "object",
				Description: "scale docs",
				Items: &Schema{Type: "string"},
			},
		},
	}
	docs := schemaToDocs(schema)
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
