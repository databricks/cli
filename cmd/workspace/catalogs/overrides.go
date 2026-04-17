package catalogs

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListCatalogsRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Type"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{blue "%s" .CatalogType}}	{{.Comment}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Name", func(c catalog.CatalogInfo) string { return c.Name }),
		tableview.Col("Type", func(c catalog.CatalogInfo) string { return string(c.CatalogType) }),
		tableview.ColMax("Comment", 40, func(c catalog.CatalogInfo) string { return c.Comment }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
