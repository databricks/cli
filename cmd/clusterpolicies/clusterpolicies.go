package clusterpolicies

import (
	cluster_policies "github.com/databricks/bricks/cmd/clusterpolicies/cluster-policies"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "clusterpolicies",
	Short: `Cluster policy limits the ability to configure clusters based on a set of rules.`,
	Long: `Cluster policy limits the ability to configure clusters based on a set of
  rules.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(cluster_policies.Cmd)
}
