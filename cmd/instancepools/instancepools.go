package instancepools

import (
	instance_pools "github.com/databricks/bricks/cmd/instancepools/instance-pools"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "instancepools",
}

func init() {

	Cmd.AddCommand(instance_pools.Cmd)
}
