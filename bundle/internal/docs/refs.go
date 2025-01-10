package main

import (
	"log"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

func isReferenceType(v *jsonschema.Schema, refs map[string]jsonschema.Schema) bool {
	if len(v.Properties) > 0 {
		return true
	}
	if v.Items != nil {
		items := resolveRefs(v.Items, refs)
		if items != nil && items.Type == "object" {
			return true
		}
	}
	props := resolveAdditionaProperties(v, refs)
	if props != nil && props.Type == "object" {
		return true
	}

	return false
}

func resolveAdditionaProperties(v *jsonschema.Schema, refs map[string]jsonschema.Schema) *jsonschema.Schema {
	if v.AdditionalProperties == nil {
		return nil
	}
	additionalProps, ok := v.AdditionalProperties.(*jsonschema.Schema)
	if !ok {
		return nil
	}
	return resolveRefs(additionalProps, refs)
}

func resolveRefs(s *jsonschema.Schema, schemas map[string]jsonschema.Schema) *jsonschema.Schema {
	node := s

	description := s.Description
	markdownDescription := s.MarkdownDescription
	examples := s.Examples

	for node.Reference != nil {
		ref := strings.TrimPrefix(*node.Reference, "#/$defs/")
		newNode, ok := schemas[ref]
		if !ok {
			log.Printf("schema %s not found", ref)
		}

		if description == "" {
			description = newNode.Description
		}
		if markdownDescription == "" {
			markdownDescription = newNode.MarkdownDescription
		}
		if len(examples) == 0 {
			examples = newNode.Examples
		}

		node = &newNode
	}

	node.Description = description
	node.MarkdownDescription = markdownDescription
	node.Examples = examples

	return node
}
