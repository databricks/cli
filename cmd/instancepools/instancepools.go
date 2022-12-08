package instancepools

import (
	instance_pools "github.com/databricks/bricks/cmd/instancepools/instance-pools"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "instancepools",
	Short: `Instance Pools API are used to create, edit, delete and list instance pools by using ready-to-use cloud instances which reduces a cluster start and auto-scaling times.`,
	Long: `Instance Pools API are used to create, edit, delete and list instance pools by
  using ready-to-use cloud instances which reduces a cluster start and
  auto-scaling times.`,
}

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

	Cmd.AddCommand(instance_pools.Cmd)
}
