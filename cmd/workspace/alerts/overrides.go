package alerts

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *sql.ListAlertsRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%s" .Id}}	{{.DisplayName}}	{{.State}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("ID", func(a sql.ListAlertsResponseAlert) string { return a.Id }),
		tableview.Col("Name", func(a sql.ListAlertsResponseAlert) string { return a.DisplayName }),
		tableview.Col("State", func(a sql.ListAlertsResponseAlert) string { return string(a.State) }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
