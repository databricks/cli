package deploy

import (
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

// WIP: will add integration test and develop this command
// NOTE: WIP, needed to add sync for
// launchCmd represents the launch command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "deploys a DAB",
	// Long:  `Reads a file and executes it on dev cluster`,
	// Args:  cobra.ExactArgs(1),
	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)

		if *remotePath != "" {
			prj.OverrideRemoteRoot(*remotePath)
		}

		targetDir, err := prj.RemoteRoot()
		if err != nil {
			return err
		}

		locker, err := CreateLocker(ctx, false, targetDir)
		if err != nil {
			return err
		}

		locker.Lock(ctx)
		defer locker.Unlock(ctx)
		return nil
	},
}

var remotePath *string

func init() {
	root.RootCmd.AddCommand(deployCmd)
	remotePath = deployCmd.Flags().String("remote-path", "", "workspace root of the project eg: /Repos/me@example.com/test-repo")
}
