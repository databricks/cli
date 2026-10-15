package cluster_policies

import (
	"github.com/databricks/cli/libs/cmdio"
	clusterpolicies "github.com/databricks/cli/libs/sdktypes/clusterpolicies/v2"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *clusterpolicies.ListPoliciesRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.PolicyId | green}}	{{.Name}}
	{{end}}`)
}

func getOverride(getCmd *cobra.Command, _ *clusterpolicies.GetPolicyRequest) {
	getCmd.Annotations["template"] = cmdio.Heredoc(`{{.Definition | pretty_json}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	getOverrides = append(getOverrides, getOverride)
}
