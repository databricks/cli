package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	md "github.com/nao1215/markdown"
)

const (
	rootFileName = "reference.md"
	rootHeader   = `---
description: Configuration reference for databricks.yml
---

# Configuration reference

This article provides reference for keys supported by <DABS> configuration (YAML). See [_](/dev-tools/bundles/index.md).

For complete bundle examples, see [_](/dev-tools/bundles/resource-examples.md) and the [bundle-examples GitHub repository](https://github.com/databricks/bundle-examples).
`
)

const (
	resourcesFileName = "resources.md"
	resourcesHeader   = `---
description: Learn about resources supported by Databricks Asset Bundles and how to configure them.
---

# <DABS> resources

<DABS> allows you to specify information about the <Databricks> resources used by the bundle in the ` + "`" + `resources` + "`" + ` mapping in the bundle configuration. See [resources mapping](/dev-tools/bundles/settings.md#resources) and [resources key reference](/dev-tools/bundles/reference.md#resources).

This article outlines supported resource types for bundles and provides details and an example for each supported type. For additional examples, see [_](/dev-tools/bundles/resource-examples.md).

## <a id="resource-types"></a> Supported resources

The following table lists supported resource types for bundles. Some resources can be created by defining them in a bundle and deploying the bundle, and some resources only support referencing an existing resource to include in the bundle.

Resources are defined using the corresponding [Databricks REST API](/api/workspace/introduction) object's create operation request payload, where the object's supported fields, expressed as YAML, are the resource's supported properties. Links to documentation for each resource's corresponding payloads are listed in the table.

.. tip:: The ` + "`" + `databricks bundle validate` + "`" + ` command returns warnings if unknown resource properties are found in bundle configuration files.


.. list-table::
    :header-rows: 1

    * - Resource
      - Create support
      - Corresponding REST API object

    * - [cluster](#cluster)
      - ✓
      - [Cluster object](/api/workspace/clusters/create)

    * - [dashboard](#dashboard)
      -
      - [Dashboard object](/api/workspace/lakeview/create)

    * - [experiment](#experiment)
      - ✓
      - [Experiment object](/api/workspace/experiments/createexperiment)

    * - [job](#job)
      - ✓
      - [Job object](/api/workspace/jobs/create)

    * - [model (legacy)](#model-legacy)
      - ✓
      - [Model (legacy) object](/api/workspace/modelregistry/createmodel)

    * - [model_serving_endpoint](#model-serving-endpoint)
      - ✓
      - [Model serving endpoint object](/api/workspace/servingendpoints/create)

    * - [pipeline](#pipeline)
      - ✓
      - [Pipeline object]](/api/workspace/pipelines/create)

    * - [quality_monitor](#quality-monitor)
      - ✓
      - [Quality monitor object](/api/workspace/qualitymonitors/create)

    * - [registered_model](#registered-model) (<UC>)
      - ✓
      - [Registered model object](/api/workspace/registeredmodels/create)

    * - [schema](#schema) (<UC>)
      - ✓
      - [Schema object](/api/workspace/schemas/create)

    * - [volume](#volume) (<UC>)
      - ✓
      - [Volume object](/api/workspace/volumes/create)
`
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
