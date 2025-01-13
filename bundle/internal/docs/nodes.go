package main

import (
	"sort"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

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
	k        string
	v        *jsonschema.Schema
	topLevel bool
}

const MapType = "Map"

func getNodes(s jsonschema.Schema, refs map[string]jsonschema.Schema, customFields map[string]bool) []rootNode {
	rootProps := []rootProp{}
	for k, v := range s.Properties {
		rootProps = append(rootProps, rootProp{k, v, true})
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

		node.Attributes = getAttributes(v.Properties, refs, customFields, k)
		rootProps = append(rootProps, extractNodes(k, v.Properties, refs, customFields)...)

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
			node.ObjectKeyAttributes = getAttributes(objectKeyType.Properties, refs, customFields, prefix)
			rootProps = append(rootProps, extractNodes(prefix, objectKeyType.Properties, refs, customFields)...)
		}

		if v.Items != nil {
			arrayItemType := resolveRefs(v.Items, refs)
			node.ArrayItemAttributes = getAttributes(arrayItemType.Properties, refs, customFields, k)
		}

		isEmpty := node.Description == "" && len(node.Attributes) == 0 && len(node.ObjectKeyAttributes) == 0 && len(node.ArrayItemAttributes) == 0
		shouldAddNode := !isEmpty || node.TopLevel
		if shouldAddNode {
			nodes = append(nodes, node)
		}
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

func getAttributes(props map[string]*jsonschema.Schema, refs map[string]jsonschema.Schema, customFields map[string]bool, prefix string) []attributeNode {
	attributes := []attributeNode{}
	for k, v := range props {
		v = resolveRefs(v, refs)
		typeString := getHumanReadableType(v.Type)
		if typeString == "" {
			typeString = "Any"
		}
		var reference string
		if isReferenceType(v, refs, customFields) {
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

func extractNodes(prefix string, props map[string]*jsonschema.Schema, refs map[string]jsonschema.Schema, customFields map[string]bool) []rootProp {
	nodes := []rootProp{}
	for k, v := range props {
		if !shouldExtract(*v.Reference, customFields) {
			continue
		}
		v = resolveRefs(v, refs)
		if v.Type == "object" || v.Type == "array" {
			nodes = append(nodes, rootProp{prefix + "." + k, v, false})
		}
	}
	return nodes
}

func getExample(v *jsonschema.Schema) string {
	examples := v.Examples
	if len(examples) == 0 {
		return ""
	}
	return examples[0].(string)
}
