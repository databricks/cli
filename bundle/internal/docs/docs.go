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

		node.Attributes = getAttributes(v.Properties, refs, k)
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
			node.ObjectKeyAttributes = getAttributes(objectKeyType.Properties, refs, k)
			rootProps = append(rootProps, extractNodes(k, objectKeyType.Properties, refs, customFields)...)
		}

		if v.Items != nil {
			arrayItemType := resolveRefs(v.Items, refs)
			node.ArrayItemAttributes = getAttributes(arrayItemType.Properties, refs, k)
			// rootProps = append(rootProps, extractNodes(k, arrayItemType.Properties, refs, customFields)...)
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

func buildMarkdown(nodes []rootNode, outputFile, header string) error {
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	m := md.NewMarkdown(f)
	m = m.PlainText(header)
	for _, node := range nodes {
		m = m.LF()
		if node.TopLevel {
			m = m.H2(node.Title)
		} else {
			m = m.H3(node.Title)
		}
		m = m.LF()

		if node.Type != "" {
			m = m.PlainText(fmt.Sprintf("**`Type: %s`**", node.Type))
			m = m.LF()
		}
		m = m.PlainText(node.Description)
		m = m.LF()

		if len(node.ObjectKeyAttributes) > 0 {
			n := removePluralForm(node.Title)
			m = m.CodeBlocks("yaml", fmt.Sprintf("%ss:\n  <%s-name>:\n    <%s-field-name>: <%s-field-value>", n, n, n, n))
			m = m.LF()
			m = buildAttributeTable(m, node.ObjectKeyAttributes)
		} else if len(node.ArrayItemAttributes) > 0 {
			m = m.LF()
			m = buildAttributeTable(m, node.ArrayItemAttributes)
		} else if len(node.Attributes) > 0 {
			m = m.LF()
			m = buildAttributeTable(m, node.Attributes)
		}

		if node.Example != "" {
			m = m.LF()
			m = m.PlainText("**Example**")
			m = m.LF()
			m = m.PlainText(node.Example)
		}
	}

	err = m.Build()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func removePluralForm(s string) string {
	if strings.HasSuffix(s, "s") {
		return strings.TrimSuffix(s, "s")
	}
	return s
}

func buildAttributeTable(m *md.Markdown, attributes []attributeNode) *md.Markdown {
	return buildCustomAttributeTable(m, attributes)

	// Rows below are useful for debugging since it renders the table in a regular markdown format

	// rows := [][]string{}
	// for _, n := range attributes {
	// 	rows = append(rows, []string{fmt.Sprintf("`%s`", n.Title), n.Type, formatDescription(n.Description)})
	// }
	// m = m.CustomTable(md.TableSet{
	// 	Header: []string{"Key", "Type", "Description"},
	// 	Rows:   rows,
	// }, md.TableOptions{AutoWrapText: false, AutoFormatHeaders: false})

	// return m
}

func formatDescription(a attributeNode) string {
	s := strings.ReplaceAll(a.Description, "\n", " ")
	if a.Reference != "" {
		if strings.HasSuffix(s, ".") {
			s += " "
		} else if s != "" {
			s += ". "
		}
		s += fmt.Sprintf("See %s.", md.Link("_", "#"+a.Reference))
	}
	return s
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
		m = m.PlainText("   * - " + fmt.Sprintf("`%s`", a.Title))
		m = m.PlainText("     - " + a.Type)
		m = m.PlainText("     - " + formatDescription(a))
		m = m.LF()
	}
	return m
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

func getAttributes(props map[string]*jsonschema.Schema, refs map[string]jsonschema.Schema, prefix string) []attributeNode {
	attributes := []attributeNode{}
	for k, v := range props {
		v = resolveRefs(v, refs)
		typeString := getHumanReadableType(v.Type)
		if typeString == "" {
			typeString = "Any"
		}
		var reference string
		if isReferenceType(v, refs) {
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

func getDescription(s *jsonschema.Schema, allowMarkdown bool) string {
	if allowMarkdown && s.MarkdownDescription != "" {
		return s.MarkdownDescription
	}
	return s.Description
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
