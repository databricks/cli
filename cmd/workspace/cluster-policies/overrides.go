package cluster_policies

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *compute.ListClusterPoliciesRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.PolicyId | green}}	{{.Name}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Policy ID", func(p compute.Policy) string { return p.PolicyId }),
		tableview.Col("Name", func(p compute.Policy) string { return p.Name }),
		tableview.Col("Default", func(p compute.Policy) string {
			if p.IsDefault {
				return "yes"
			}
			return ""
		}),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func getOverride(getCmd *cobra.Command, _ *compute.GetClusterPolicyRequest) {
	getCmd.Annotations["template"] = cmdio.Heredoc(`{{.Definition | pretty_json}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	getOverrides = append(getOverrides, getOverride)
}
