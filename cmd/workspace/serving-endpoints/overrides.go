package serving_endpoints

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%s" .Name}}	{{if .State}}{{.State.Ready}}{{end}}	{{.Creator}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Name", func(e serving.ServingEndpoint) string { return e.Name }),
		tableview.Col("State", func(e serving.ServingEndpoint) string {
			if e.State != nil {
				return string(e.State.Ready)
			}
			return ""
		}),
		tableview.Col("Creator", func(e serving.ServingEndpoint) string { return e.Creator }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
