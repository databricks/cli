package schemas

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListSchemasRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Owner"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{.Owner|cyan}}	{{.Comment}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Full Name", Extract: func(v any) string {
			return v.(catalog.SchemaInfo).FullName
		}},
		{Header: "Owner", Extract: func(v any) string {
			return v.(catalog.SchemaInfo).Owner
		}},
		{Header: "Comment", MaxWidth: 40, Extract: func(v any) string {
			return v.(catalog.SchemaInfo).Comment
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
