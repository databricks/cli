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
	Reference   string
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

func getNodes(s jsonschema.Schema, refs map[string]*jsonschema.Schema, customFields map[string]bool) []rootNode {
	rootProps := []rootProp{}
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
		v = resolveRefs(v, refs)
		node := rootNode{
			Title:       k,
			Description: getDescription(v, item.topLevel),
			TopLevel:    item.topLevel,
			Example:     getExample(v),
			Type:        getHumanReadableType(v.Type),
		}

		node.Attributes = getAttributes(v.Properties, refs, customFields, k, item.circular)
		if !item.circular {
			rootProps = append(rootProps, extractNodes(k, v.Properties, refs, customFields)...)
		}

		additionalProps, ok := v.AdditionalProperties.(*jsonschema.Schema)
		if ok {
			objectKeyType := resolveRefs(additionalProps, refs)
			d := getDescription(objectKeyType, true)
			if d != "" {
				node.Description = d
			}
			if len(node.Example) == 0 {
				node.Example = getExample(objectKeyType)
			}
			prefix := k + ".<name>"
			node.ObjectKeyAttributes = getAttributes(objectKeyType.Properties, refs, customFields, prefix, item.circular)
			if !item.circular {
				rootProps = append(rootProps, extractNodes(prefix, objectKeyType.Properties, refs, customFields)...)
			}
		}

		if v.Items != nil {
			arrayItemType := resolveRefs(v.Items, refs)
			node.ArrayItemAttributes = getAttributes(arrayItemType.Properties, refs, customFields, k, item.circular)
			if !item.circular {
				rootProps = append(rootProps, extractNodes(k, arrayItemType.Properties, refs, customFields)...)
			}
		}

		nodes = append(nodes, node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Title < nodes[j].Title
	})
	return nodes
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

func getAttributes(props, refs map[string]*jsonschema.Schema, customFields map[string]bool, prefix string, circular bool) []attributeNode {
	attributes := []attributeNode{}
	for k, v := range props {
		v = resolveRefs(v, refs)
		typeString := getHumanReadableType(v.Type)
		if typeString == "" {
			typeString = "Any"
		}
		var reference string
		if isReferenceType(v, refs, customFields) && !circular {
			reference = prefix + "." + k
		}
		attributes = append(attributes, attributeNode{
			Title:       k,
			Type:        typeString,
			Description: getDescription(v, true),
			Reference:   reference,
		})
	}
	sort.Slice(attributes, func(i, j int) bool {
		return attributes[i].Title < attributes[j].Title
	})
	return attributes
}

func getDescription(s *jsonschema.Schema, allowMarkdown bool) string {
	if allowMarkdown && s.MarkdownDescription != "" {
		return s.MarkdownDescription
	}
	return s.Description
}

func shouldExtract(ref string, customFields map[string]bool) bool {
	if i := strings.Index(ref, "github.com"); i >= 0 {
		ref = ref[i:]
	}
	_, isCustomField := customFields[ref]
	return isCustomField
}

func extractNodes(prefix string, props, refs map[string]*jsonschema.Schema, customFields map[string]bool) []rootProp {
	nodes := []rootProp{}
	for k, v := range props {
		if !shouldExtract(*v.Reference, customFields) {
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
	examples := v.Examples
	if len(examples) == 0 {
		return ""
	}
	return examples[0].(string)
}
