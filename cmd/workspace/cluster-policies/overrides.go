package cluster_policies

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *compute.ListClusterPoliciesRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.PolicyId | green}}	{{.Name}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Policy ID", Extract: func(v any) string {
			return v.(compute.Policy).PolicyId
		}},
		{Header: "Name", Extract: func(v any) string {
			return v.(compute.Policy).Name
		}},
		{Header: "Default", Extract: func(v any) string {
			if v.(compute.Policy).IsDefault {
				return "yes"
			}
			return ""
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func getOverride(getCmd *cobra.Command, _ *compute.GetClusterPolicyRequest) {
	getCmd.Annotations["template"] = cmdio.Heredoc(`{{.Definition | pretty_json}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	getOverrides = append(getOverrides, getOverride)
}
