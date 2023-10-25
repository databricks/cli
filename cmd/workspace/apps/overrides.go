package apps

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

func createOverride(cmd *cobra.Command, deployReq *serving.DeployAppRequest) {
	var manifestYaml flags.YamlFlag
	var resourcesYaml flags.YamlFlag

	// TODO: short flags
	cmd.Flags().Var(&manifestYaml, "manifest", `path/to/manifest.yaml`)
	cmd.Flags().Var(&resourcesYaml, "resources", `path/to/resources.yaml`)

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("manifest") {
			err = manifestYaml.Unmarshal(&deployReq.Manifest)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in YAML format by specifying the --manifest flag")
		}

		if cmd.Flags().Changed("resources") {
			err = resourcesYaml.Unmarshal(&deployReq.Resources)
			if err != nil {
				return err
			}
		}
		response, err := w.Apps.Create(ctx, *deployReq)

		if err != nil {
			return err
		}

		return cmdio.Render(ctx, response)
	}
}

func init() {
	createOverrides = append(createOverrides, createOverride)
}
