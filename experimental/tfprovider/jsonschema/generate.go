package jsonschema

import (
	"encoding/json"
	"fmt"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func processAttributeType(typ cty.Type) *Schema {
	var out Schema

	switch {
	case typ.IsPrimitiveType():
		switch {
		case typ.Equals(cty.Bool):
			out.Type = "boolean"
		case typ.Equals(cty.Number):
			out.Type = "integer"
		case typ.Equals(cty.String):
			out.Type = "string"
		default:
			panic("No idea what to do for: " + typ.FriendlyName())
		}
	case typ.IsMapType():
		out.Type = "object"
	case typ.IsSetType():
		out.Type = "array"
		out.Items = processAttributeType(*typ.SetElementType())
	case typ.IsListType():
		out.Type = "array"
		out.Items = processAttributeType(*typ.ListElementType())
	default:
		panic("No idea what to do for: " + typ.FriendlyName())
	}

	return &out
}

func processSchemaAttribute(in *tfjson.SchemaAttribute) *Schema {
	out := processAttributeType(in.AttributeType)
	out.Description = in.Description
	return out
}

func processBlock(in *tfjson.SchemaBlock) *Schema {
	var out = Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for k, v := range in.Attributes {
		if v.Deprecated {
			continue
		}

		if v.Required {
			out.Required = append(out.Required, k)
		}

		out.Properties[k] = processSchemaAttribute(v)
	}

	for k, v := range in.NestedBlocks {
		out.Properties[k] = processBlock(v.Block)
	}

	out.Description = in.Description
	return &out
}

func toObject(in *tfjson.Schema) *Schema {
	return processBlock(in.Block)
}

func Generate(schema *tfjson.ProviderSchema) error {
	var resources = Schema{
		Type:       "object",
		Title:      "resources",
		Properties: make(map[string]*Schema),
	}

	for k, v := range schema.ResourceSchemas {
		k = strings.TrimPrefix(k, "databricks_")
		resources.Properties[k] = &Schema{
			Type:                 "object",
			AdditionalProperties: toObject(v),
		}
	}

	var out = Schema{
		Type:  "object",
		Title: "resource file",

		Properties: map[string]*Schema{
			"environments": {
				Type: "object",
				AdditionalProperties: &Schema{
					Type: "object",
					Properties: map[string]*Schema{
						"resources": &resources,
					},
				},
			},
			"resources": &resources,
		},
	}

	raw, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(raw))

	return nil
}
