package instance_pools

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.InstancePoolId|green}}	{{.InstancePoolName}}	{{.NodeTypeId}}	{{.State}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Pool ID", Extract: func(v any) string {
			return v.(compute.InstancePoolAndStats).InstancePoolId
		}},
		{Header: "Name", Extract: func(v any) string {
			return v.(compute.InstancePoolAndStats).InstancePoolName
		}},
		{Header: "Node Type", Extract: func(v any) string {
			return v.(compute.InstancePoolAndStats).NodeTypeId
		}},
		{Header: "State", Extract: func(v any) string {
			return string(v.(compute.InstancePoolAndStats).State)
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
