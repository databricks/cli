package main

import (
	"testing"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/stretchr/testify/assert"
)

func TestBuildNodes_ChildExpansion(t *testing.T) {
	tests := []struct {
		name      string
		schema    jsonschema.Schema
		refs      map[string]*jsonschema.Schema
		ownFields map[string]bool
		wantNodes []rootNode
	}{
		{
			name: "array expansion",
			schema: jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"list": {
						Type: "array",
						Items: &jsonschema.Schema{
							Type: "object",
							Properties: map[string]*jsonschema.Schema{
								"listSub": {Reference: strPtr("#/$defs/github.com/listSub")},
							},
						},
					},
				},
			},
			refs: map[string]*jsonschema.Schema{
				"github.com/listSub": {Type: "array", Items: &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{"subField": {Type: "string"}}}},
			},
			ownFields: map[string]bool{"github.com/listSub": true},
			wantNodes: []rootNode{
				{
					Title:    "list",
					TopLevel: true,
					Type:     "Sequence",
					ArrayItemAttributes: []attributeNode{
						{Title: "listSub", Type: "Sequence", Link: "list.listSub"},
					},
				},
				{
					Title: "list.listSub",
					Type:  "Sequence",
					ArrayItemAttributes: []attributeNode{
						{Title: "subField", Type: "String"},
					},
				},
			},
		},
		{
			name: "map expansion",
			schema: jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"myMap": {
						Type: "object",
						AdditionalProperties: &jsonschema.Schema{
							Reference: strPtr("#/$defs/github.com/myMap"),
							Properties: map[string]*jsonschema.Schema{
								"mapSub": {Type: "object", Reference: strPtr("#/$defs/github.com/mapSub")},
							},
						},
					},
				},
			},
			refs: map[string]*jsonschema.Schema{
				"github.com/myMap": {
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"mapSub": {Type: "boolean", Reference: strPtr("#/$defs/github.com/mapSub")},
					},
				},
				"github.com/mapSub": {
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"deepSub": {Type: "boolean"},
					},
				},
			},
			ownFields: map[string]bool{
				"github.com/myMap":  true,
				"github.com/mapSub": true,
			},
			wantNodes: []rootNode{
				{
					Title:    "myMap",
					TopLevel: true,
					Type:     "Map",
					ObjectKeyAttributes: []attributeNode{
						{Title: "mapSub", Type: "Map", Link: "myMap._name_.mapSub"},
					},
				},
				{
					Title: "myMap._name_.mapSub",
					Type:  "Map",
					Attributes: []attributeNode{
						{Title: "deepSub", Type: "Boolean"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNodes(tt.schema, tt.refs, tt.ownFields)
			assert.Equal(t, tt.wantNodes, got)
		})
	}
}

func TestDeprecatedFields(t *testing.T) {
	s := jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"deprecatedField": {Deprecated: true},
			"notDeprecatedField": {
				Properties: map[string]*jsonschema.Schema{
					"nestedDeprecatedField": {
						Deprecated:  true,
						Description: "nested description",
						Extension: jsonschema.Extension{
							DeprecationMessage: "nested deprecation message",
						},
					},
					"nestedNotDeprecatedField": {},
				},
			},
		},
	}
	nodes := buildNodes(s, nil, nil)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "notDeprecatedField", nodes[0].Title)

	assert.Len(t, nodes[0].Attributes, 2)
	assert.Equal(t, "nested deprecation message", nodes[0].Attributes[0].Description)
}

func strPtr(s string) *string {
	return &s
}
