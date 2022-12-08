package clusters

import (
	"github.com/databricks/bricks/cmd/clusters/clusters"
	instance_profiles "github.com/databricks/bricks/cmd/clusters/instance-profiles"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "clusters",
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(clusters.Cmd)
	Cmd.AddCommand(instance_profiles.Cmd)
}
