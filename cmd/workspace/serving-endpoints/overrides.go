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
		{Header: "Name", Extract: func(v any) string {
			return v.(serving.ServingEndpoint).Name
		}},
		{Header: "State", Extract: func(v any) string {
			if v.(serving.ServingEndpoint).State != nil {
				return string(v.(serving.ServingEndpoint).State.Ready)
			}
			return ""
		}},
		{Header: "Creator", Extract: func(v any) string {
			return v.(serving.ServingEndpoint).Creator
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
