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
		tableview.Col("Pool ID", func(p compute.InstancePoolAndStats) string { return p.InstancePoolId }),
		tableview.Col("Name", func(p compute.InstancePoolAndStats) string { return p.InstancePoolName }),
		tableview.Col("Node Type", func(p compute.InstancePoolAndStats) string { return p.NodeTypeId }),
		tableview.Col("State", func(p compute.InstancePoolAndStats) string { return string(p.State) }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
