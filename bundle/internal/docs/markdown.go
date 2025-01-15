package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	md "github.com/nao1215/markdown"
)

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

	err = m.Build()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func pickLastWord(s string) string {
	words := strings.Split(s, ".")
	return words[len(words)-1]
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
