package main

import (
	"sort"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

// rootNode is an intermediate representation of resolved JSON-schema item that is used to generate documentation
// Every schema node goes follows this conversion `JSON-schema -> rootNode -> markdown text`
type rootNode struct {
	Title               string
	Description         string
	Attributes          []attributeNode
	Example             string
	ObjectKeyAttributes []attributeNode
	ArrayItemAttributes []attributeNode
	TopLevel            bool
	Type                string
}

type attributeNode struct {
	Title       string
	Type        string
	Description string
	Link        string
}

type rootProp struct {
	// k is the name of the property
	k string
	// v is the corresponding json-schema node
	v *jsonschema.Schema
	// topLevel is true only for direct properties of the schema of root type (e.g. config.Root or config.Resources)
	// Example: config.Root has .
	topLevel bool
	// circular indicates if property was added by recursive type, e.g. task.for_each_task.task.for_each_task
	// These entries don't expand further and don't add any new nodes from their properties
	circular bool
}

const MapType = "Map"

// buildNodes converts JSON-schema to a flat list of rootNode items that are then used to generate markdown documentation
// It recursively traverses the schema expanding the resulting list with new items for every properties of nodes `object` and `array` type
func buildNodes(s jsonschema.Schema, refs map[string]*jsonschema.Schema, ownFields map[string]bool) []rootNode {
	var rootProps []rootProp
	for k, v := range s.Properties {
		rootProps = append(rootProps, rootProp{k, v, true, false})
	}
	nodes := make([]rootNode, 0, len(rootProps))
	visited := make(map[string]bool)

	for i := 0; i < len(rootProps); i++ {
		item := rootProps[i]
		k := item.k
		v := item.v

		if visited[k] {
			continue
		}
		visited[k] = true

		if v.Deprecated {
			continue
		}

		v = resolveRefs(v, refs)
		node := rootNode{
			Title:       k,
			Description: getDescription(v),
			TopLevel:    item.topLevel,
			Example:     getExample(v),
			Type:        getHumanReadableType(v.Type),
		}

		hasProperties := len(v.Properties) > 0
		if hasProperties {
			node.Attributes = getAttributes(v.Properties, refs, ownFields, k, item.circular)
		}

		mapValueType := getMapValueType(v, refs)
		if mapValueType != nil {
			d := getDescription(mapValueType)
			if d != "" {
				node.Description = d
			}
			if node.Example == "" {
				node.Example = getExample(mapValueType)
			}
			node.ObjectKeyAttributes = getAttributes(mapValueType.Properties, refs, ownFields, getMapKeyPrefix(k), item.circular)
		}

		arrayItemType := resolveRefs(v.Items, refs)
		if arrayItemType != nil {
			node.ArrayItemAttributes = getAttributes(arrayItemType.Properties, refs, ownFields, k, item.circular)
		}

		nodes = append(nodes, node)

		// Whether we should add new root props from the children of the current JSON-schema node to include their definitions to this document
		shouldAddNewProps := !item.circular
		if shouldAddNewProps {
			var newProps []rootProp
			// Adds node with definition for the properties. Example:
			// bundle:
			//  prop-name: <value>
			if hasProperties {
				newProps = append(newProps, extractNodes(k, v.Properties, refs, ownFields)...)
			}

			// Adds node with definition for the type of array item. Example:
			// permissions:
			//  - <item>
			if arrayItemType != nil {
				newProps = append(newProps, extractNodes(k, arrayItemType.Properties, refs, ownFields)...)
			}
			// Adds node with definition for the type of the Map value. Example:
			// targets:
			//   <key>: <value>
			if mapValueType != nil {
				newProps = append(newProps, extractNodes(getMapKeyPrefix(k), mapValueType.Properties, refs, ownFields)...)
			}

			rootProps = append(rootProps, newProps...)
		}
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Title < nodes[j].Title
	})
	return nodes
}

func getMapValueType(v *jsonschema.Schema, refs map[string]*jsonschema.Schema) *jsonschema.Schema {
	additionalProps, ok := v.AdditionalProperties.(*jsonschema.Schema)
	if ok {
		return resolveRefs(additionalProps, refs)
	}
	return nil
}

const (
	nameField           = "name"
	nameFieldWithFormat = "_name_"
)

func getMapKeyPrefix(s string) string {
	return s + "." + nameFieldWithFormat
}

func removePluralForm(s string) string {
	if strings.HasSuffix(s, "s") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func getHumanReadableType(t jsonschema.Type) string {
	typesMapping := map[string]string{
		"string":  "String",
		"integer": "Integer",
		"boolean": "Boolean",
		"array":   "Sequence",
		"object":  "Map",
	}
	return typesMapping[string(t)]
}

func getAttributes(props, refs map[string]*jsonschema.Schema, ownFields map[string]bool, prefix string, circular bool) []attributeNode {
	var attributes []attributeNode
	for k, v := range props {
		v = resolveRefs(v, refs)
		typeString := getHumanReadableType(v.Type)
		if typeString == "" {
			typeString = "Any"
		}
		var reference string
		if isReferenceType(v, refs, ownFields) && !circular && !v.Deprecated {
			reference = prefix + "." + k
		}
		attributes = append(attributes, attributeNode{
			Title:       k,
			Type:        typeString,
			Description: getDescription(v),
			Link:        reference,
		})
	}
	sort.Slice(attributes, func(i, j int) bool {
		return attributes[i].Title < attributes[j].Title
	})
	return attributes
}

func getDescription(s *jsonschema.Schema) string {
	if s.DeprecationMessage != "" {
		return s.DeprecationMessage
	}
	if s.MarkdownDescription != "" {
		return s.MarkdownDescription
	}
	return s.Description
}

func shouldExtract(ref string, ownFields map[string]bool) bool {
	if i := strings.Index(ref, "github.com"); i >= 0 {
		ref = ref[i:]
	}
	_, isCustomField := ownFields[ref]
	return isCustomField
}

// extractNodes returns a list of rootProp items for all properties of the json-schema node that should be extracted based on context
// E.g. we extract all propert
func extractNodes(prefix string, props, refs map[string]*jsonschema.Schema, ownFields map[string]bool) []rootProp {
	var nodes []rootProp
	for k, v := range props {
		if v.Reference != nil && !shouldExtract(*v.Reference, ownFields) {
			continue
		}
		v = resolveRefs(v, refs)
		if v.Type == "object" || v.Type == "array" {
			nodes = append(nodes, rootProp{prefix + "." + k, v, false, isCycleField(k)})
		}
	}
	return nodes
}

func isCycleField(field string) bool {
	return field == "for_each_task"
}

func getExample(v *jsonschema.Schema) string {
	examples := getExamples(v.Examples)
	if len(examples) == 0 {
		return ""
	}
	return examples[0]
}
