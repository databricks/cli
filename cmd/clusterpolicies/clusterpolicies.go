package clusterpolicies

import (
	cluster_policies "github.com/databricks/bricks/cmd/clusterpolicies/cluster-policies"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "clusterpolicies",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(cluster_policies.Cmd)
}
