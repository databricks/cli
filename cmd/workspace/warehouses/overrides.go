package warehouses

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *sql.ListWarehousesRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Size"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{.ClusterSize|cyan}}	{{if eq .State "RUNNING"}}{{"RUNNING"|green}}{{else if eq .State "STOPPED"}}{{"STOPPED"|red}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("ID", func(e sql.EndpointInfo) string { return e.Id }),
		tableview.Col("Name", func(e sql.EndpointInfo) string { return e.Name }),
		tableview.Col("Size", func(e sql.EndpointInfo) string { return e.ClusterSize }),
		tableview.Col("State", func(e sql.EndpointInfo) string { return string(e.State) }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
