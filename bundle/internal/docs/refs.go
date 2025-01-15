package main

import (
	"log"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

func isReferenceType(v *jsonschema.Schema, refs map[string]jsonschema.Schema, customFields map[string]bool) bool {
	if v.Type != "object" && v.Type != "array" {
		return false
	}
	if len(v.Properties) > 0 {
		return true
	}
	if v.Items != nil {
		items := resolveRefs(v.Items, refs)
		if items != nil && items.Type == "object" {
			return true
		}
	}
	props := resolveAdditionalProperties(v)
	if !isInOwnFields(props, customFields) {
		return false
	}
	if props != nil {
		propsResolved := resolveRefs(props, refs)
		return propsResolved.Type == "object"
	}

	return false
}

func isInOwnFields(node *jsonschema.Schema, customFields map[string]bool) bool {
	if node != nil && node.Reference != nil {
		return customFields[getRefType(node)]
	}
	return true
}

func resolveAdditionalProperties(v *jsonschema.Schema) *jsonschema.Schema {
	if v.AdditionalProperties == nil {
		return nil
	}
	additionalProps, ok := v.AdditionalProperties.(*jsonschema.Schema)
	if !ok {
		return nil
	}
	return additionalProps
}

func resolveRefs(s *jsonschema.Schema, schemas map[string]jsonschema.Schema) *jsonschema.Schema {
	node := s

	description := s.Description
	markdownDescription := s.MarkdownDescription
	examples := s.Examples

	for node.Reference != nil {
		ref := getRefType(node)
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

func getRefType(node *jsonschema.Schema) string {
	if node.Reference == nil {
		return ""
	}
	return strings.TrimPrefix(*node.Reference, "#/$defs/")
}
