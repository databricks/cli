package deploy

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// launchCmd represents the launch command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys a DAB",
	// Long:  `Reads a file and executes it on dev cluster`,
	// Args:  cobra.ExactArgs(1),
	PreRunE: project.Configure,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		deployMutex := &DeployMutex{
			User: "shreyas.goenka@databricks.com",
			// TODO: Adjust this using a command line arguement
			IsForced: false,
			// TODO: pass through cmd line arg
			ProjectRoot: "/Repos/shreyas.goenka@databricks.com/test-dbx",
		}
		deployMutex.Lock(ctx)
		defer deployMutex.Unlock(ctx)
	},
}

func init() {
	root.RootCmd.AddCommand(deployCmd)
}
