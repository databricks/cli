package clusterpolicies

import (
	cluster_policies "github.com/databricks/bricks/cmd/clusterpolicies/cluster-policies"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "clusterpolicies",
}

func init() {

	Cmd.AddCommand(cluster_policies.Cmd)
}
