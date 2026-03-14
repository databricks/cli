package catalogs

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListCatalogsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Type"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{blue "%s" .CatalogType}}	{{.Comment}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Name", Extract: func(v any) string {
			return v.(catalog.CatalogInfo).Name
		}},
		{Header: "Type", Extract: func(v any) string {
			return string(v.(catalog.CatalogInfo).CatalogType)
		}},
		{Header: "Comment", MaxWidth: 40, Extract: func(v any) string {
			return v.(catalog.CatalogInfo).Comment
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
