package main

import (
	"log"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

func isReferenceType(v *jsonschema.Schema, refs map[string]*jsonschema.Schema, ownFields map[string]bool) bool {
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
	if !isInOwnFields(props, ownFields) {
		return false
	}
	if props != nil {
		propsResolved := resolveRefs(props, refs)
		return propsResolved.Type == "object"
	}

	return false
}

func isInOwnFields(node *jsonschema.Schema, ownFields map[string]bool) bool {
	if node != nil && node.Reference != nil {
		return ownFields[getRefType(node)]
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

func resolveRefs(s *jsonschema.Schema, schemas map[string]*jsonschema.Schema) *jsonschema.Schema {
	if s == nil {
		return nil
	}

	node := s
	description := s.Description
	markdownDescription := s.MarkdownDescription
	examples := getExamples(s.Examples)
	deprecated := s.Deprecated
	deprecationMessage := s.DeprecationMessage

	for node.Reference != nil {
		ref := getRefType(node)
		newNode, ok := schemas[ref]
		if !ok {
			log.Printf("schema %s not found", ref)
			break
		}

		if description == "" {
			description = newNode.Description
		}
		if markdownDescription == "" {
			markdownDescription = newNode.MarkdownDescription
		}
		if len(examples) == 0 {
			examples = getExamples(newNode.Examples)
		}

		node = newNode
	}

	newNode := *node
	newNode.Description = description
	newNode.MarkdownDescription = markdownDescription
	newNode.Examples = examples
	newNode.Deprecated = deprecated
	newNode.DeprecationMessage = deprecationMessage

	return &newNode
}

func getExamples(examples any) []string {
	typedExamples, ok := examples.([]string)
	if !ok {
		return []string{}
	}
	return typedExamples
}

func getRefType(node *jsonschema.Schema) string {
	if node.Reference == nil {
		return ""
	}
	return strings.TrimPrefix(*node.Reference, "#/$defs/")
}
