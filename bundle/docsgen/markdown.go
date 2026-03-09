package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func buildMarkdown(nodes []rootNode, outputFile, header string) error {
	m := newMardownRenderer()
	m = m.PlainText(header)
	for _, node := range nodes {
		m = m.LF()
		title := node.Title
		if node.TopLevel {
			m = m.H2(title)
		} else {
			m = m.H3(title)
		}
		m = m.LF()

		if node.Type != "" {
			m = m.PlainText(fmt.Sprintf("**`Type: %s`**", node.Type))
			m = m.LF()
		}
		m = m.PlainText(node.Description)
		m = m.LF()

		if len(node.ObjectKeyAttributes) > 0 {
			n := pickLastWord(node.Title)
			n = removePluralForm(n)
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

	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}
	_, err = f.WriteString(m.String())
	if err != nil {
		log.Fatal(err)
	}
	return f.Close()
}

func pickLastWord(s string) string {
	words := strings.Split(s, ".")
	return words[len(words)-1]
}

// Build a custom table which we use in Databricks website
func buildAttributeTable(m *markdownRenderer, attributes []attributeNode) *markdownRenderer {
	m = m.LF()
	m = m.PlainText(":::list-table")
	m = m.LF()

	m = m.PlainText("- - Key")
	m = m.PlainText("  - Type")
	m = m.PlainText("  - Description")
	m = m.LF()

	for _, a := range attributes {
		m = m.PlainText("- - " + fmt.Sprintf("`%s`", a.Title))
		m = m.PlainText("  - " + a.Type)
		m = m.PlainText("  - " + formatDescription(a))
		m = m.LF()
	}

	m = m.PlainText(":::")
	m = m.LF()

	return m
}

func formatDescription(a attributeNode) string {
	s := strings.ReplaceAll(a.Description, "\n", " ")
	if a.Link != "" {
		if strings.HasSuffix(s, ".") {
			s += " "
		} else if s != "" {
			s += ". "
		}
		s += fmt.Sprintf("See [\\_](#%s).", cleanAnchor(a.Link))
	}
	return s
}

// Docs framework does not allow special characters in anchor links and strip them out by default
// We need to clean them up to make sure the links pass the validation
func cleanAnchor(s string) string {
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, nameFieldWithFormat, nameField)
	return s
}
