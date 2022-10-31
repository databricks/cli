package build

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Build and deploy artifacts",
	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)
		// https://github.com/databrickslabs/mosaic - both maven and python
		// https://github.com/databrickslabs/arcuate - only python, no DBR needed, but has notebooks
		all, err := prj.LocalArtifacts(ctx)
		if err != nil {
			return err
		}
		if len(all) == 0 {
			return fmt.Errorf("nothing to deploy")
		}
		err = ui.SpinStages(ctx, []ui.Stage{
			{InProgress: "Preparing", Callback: prj.Prepare, Complete: "Prepared!"},
			{InProgress: "Building", Callback: prj.Build, Complete: "Built!"},
			{InProgress: "Uploading", Callback: prj.Upload, Complete: "Uploaded!"},
			{InProgress: "Installing", Callback: prj.Install, Complete: "Installed!"},
		})
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(buildCmd)
}
