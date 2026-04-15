package tables

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListTablesRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Table Type"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{blue "%s" .TableType}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Full Name", Extract: func(v any) string {
			return v.(catalog.TableInfo).FullName
		}},
		{Header: "Table Type", Extract: func(v any) string {
			return string(v.(catalog.TableInfo).TableType)
		}},
	}

	listCmd.SetContext(tableview.SetTableConfig(listCmd.Context(), &tableview.TableConfig{Columns: columns}))
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
