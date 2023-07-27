package cluster_policies

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *compute.ListClusterPoliciesRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.PolicyId | green}}	{{.Name}}
	{{end}}`)
}

func getOverride(getCmd *cobra.Command, _ *compute.GetClusterPolicyRequest) {
	getCmd.Annotations["template"] = cmdio.Heredoc(`{{.Definition | pretty_json}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	getOverrides = append(getOverrides, getOverride)
}
