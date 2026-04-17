package schemas

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListSchemasRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Owner"}}	{{header "Comment"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.FullName|green}}	{{.Owner|cyan}}	{{.Comment}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Full Name", func(s catalog.SchemaInfo) string { return s.FullName }),
		tableview.Col("Owner", func(s catalog.SchemaInfo) string { return s.Owner }),
		tableview.ColMax("Comment", 40, func(s catalog.SchemaInfo) string { return s.Comment }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
