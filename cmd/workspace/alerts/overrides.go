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
		{Header: "ID", Extract: func(v any) string {
			return v.(sql.ListAlertsResponseAlert).Id
		}},
		{Header: "Name", Extract: func(v any) string {
			return v.(sql.ListAlertsResponseAlert).DisplayName
		}},
		{Header: "State", Extract: func(v any) string {
			return string(v.(sql.ListAlertsResponseAlert).State)
		}},
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
