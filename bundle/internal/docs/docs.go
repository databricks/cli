package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"

	md "github.com/nao1215/markdown"
)

type rootNode struct {
	Title               string
	Description         string
	Attributes          []attributeNode
	Example             string
	ObjectKeyAttributes []attributeNode
	ArrayItemAttributes []attributeNode
	TopLevel            bool
}

type attributeNode struct {
	Title       string
	Type        string
	Description string
}

type rootProp struct {
	k        string
	v        *jsonschema.Schema
	topLevel bool
}

func getNodes(s jsonschema.Schema, refs map[string]jsonschema.Schema, a annotationFile) []rootNode {
	rootProps := []rootProp{}
	for k, v := range s.Properties {
		rootProps = append(rootProps, rootProp{k, v, true})
	}
	nodes := make([]rootNode, 0, len(rootProps))

	for i := 0; i < len(rootProps); i++ {
		k := rootProps[i].k
		v := rootProps[i].v
		v = resolveRefs(v, refs)
		node := rootNode{
			Title:       k,
			Description: getDescription(v),
			TopLevel:    rootProps[i].topLevel,
		}

		node.Attributes = getAttributes(v.Properties, refs)
		rootProps = append(rootProps, extractNodes(k, v.Properties, refs, a)...)

		additionalProps, ok := v.AdditionalProperties.(*jsonschema.Schema)
		if ok {
			objectKeyType := resolveRefs(additionalProps, refs)
			node.ObjectKeyAttributes = getAttributes(objectKeyType.Properties, refs)
			rootProps = append(rootProps, extractNodes(k, objectKeyType.Properties, refs, a)...)
		}

		if v.Items != nil {
			arrayItemType := resolveRefs(v.Items, refs)
			node.ArrayItemAttributes = getAttributes(arrayItemType.Properties, refs)
		}

		isEmpty := len(node.Attributes) == 0 && len(node.ObjectKeyAttributes) == 0 && len(node.ArrayItemAttributes) == 0
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

func buildMarkdown(nodes []rootNode, outputFile string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m := md.NewMarkdown(f)
	for _, node := range nodes {
		m = m.LF()
		if node.TopLevel {
			m = m.H2(node.Title)
		} else {
			m = m.H3(node.Title)
		}
		m = m.PlainText(node.Description)
		m = m.LF()

		if len(node.ObjectKeyAttributes) > 0 {
			m = buildAttributeTable(m, []attributeNode{
				{Title: fmt.Sprintf("<%s-entry-name>", node.Title), Type: "Map", Description: fmt.Sprintf("Item of the `%s` map", node.Title)},
			})
			m = m.PlainText("Each item has the following attributes:")
			m = m.LF()
			m = buildAttributeTable(m, node.ObjectKeyAttributes)
		} else if len(node.ArrayItemAttributes) > 0 {
			m = m.PlainTextf("Each item of `%s` has the following attributes:", node.Title)
			m = m.LF()
			m = buildAttributeTable(m, node.ArrayItemAttributes)
		} else if len(node.Attributes) > 0 {
			m = m.H4("Attributes")
			m = m.LF()
			m = buildAttributeTable(m, node.Attributes)
		}
	}

	err = m.Build()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func buildAttributeTable(m *md.Markdown, attributes []attributeNode) *md.Markdown {
	return buildCustomAttributeTable(m, attributes)
	rows := [][]string{}
	for _, n := range attributes {
		rows = append(rows, []string{fmt.Sprintf("`%s`", n.Title), n.Type, formatDescription(n.Description)})
	}
	m = m.CustomTable(md.TableSet{
		Header: []string{"Key", "Type", "Description"},
		Rows:   rows,
	}, md.TableOptions{AutoWrapText: false, AutoFormatHeaders: false})

	return m
}

func formatDescription(s string) string {
	return strings.ReplaceAll(s, "\n", " ")
}

// Build a custom table which we use in Databricks website
func buildCustomAttributeTable(m *md.Markdown, attributes []attributeNode) *md.Markdown {
	m = m.LF()
	m = m.PlainText(".. list-table::")
	m = m.PlainText("   :header-rows: 1")
	m = m.LF()

	m = m.PlainText("   * - Key")
	m = m.PlainText("     - Type")
	m = m.PlainText("     - Description")
	m = m.LF()

	for _, a := range attributes {
		m = m.PlainText("   * - " + a.Title)
		m = m.PlainText("     - " + a.Type)
		m = m.PlainText("     - " + formatDescription(a.Description))
		m = m.LF()
	}
	return m
}

func getAttributes(props map[string]*jsonschema.Schema, refs map[string]jsonschema.Schema) []attributeNode {
	typesMapping := map[string]string{
		"string":  "String",
		"integer": "Integer",
		"boolean": "Boolean",
		"array":   "Sequence",
		"object":  "Map",
	}

	attributes := []attributeNode{}
	for k, v := range props {
		v = resolveRefs(v, refs)
		typeString := typesMapping[string(v.Type)]
		if typeString == "" {
			typeString = "Any"
		}
		attributes = append(attributes, attributeNode{
			Title:       k,
			Type:        typeString,
			Description: getDescription(v),
		})
	}
	sort.Slice(attributes, func(i, j int) bool {
		return attributes[i].Title < attributes[j].Title
	})
	return attributes
}

func getDescription(s *jsonschema.Schema) string {
	if s.MarkdownDescription != "" {
		return s.MarkdownDescription
	}
	return s.Description
}

func resolveRefs(s *jsonschema.Schema, schemas map[string]jsonschema.Schema) *jsonschema.Schema {
	node := s

	description := s.Description
	markdownDescription := s.MarkdownDescription

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

		node = &newNode
	}

	node.Description = description
	node.MarkdownDescription = markdownDescription

	return node
}

func extractNodes(prefix string, props map[string]*jsonschema.Schema, refs map[string]jsonschema.Schema, a annotationFile) []rootProp {
	nodes := []rootProp{}
	for k, v := range props {
		v = resolveRefs(v, refs)
		if v.Type == "object" {
			nodes = append(nodes, rootProp{prefix + "." + k, v, false})
		}
		v.MarkdownDescription = ""
	}
	return nodes
}
