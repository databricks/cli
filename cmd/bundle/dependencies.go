package bundle

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Terraform *terraform.TerraformMetadata `json:"terraform"`
}

func newDependenciesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "internal-dependencies",
		Short: "Prints CLI terraform dependencies required for the bundle namespace",
		Args:  root.NoArgs,
		Annotations: map[string]string{
			"template": `Terraform version: {{.Version}}
Terraform URL: https://releases.hashicorp.com/terraform/{{.Version}}

Databricks Terraform Provider version: {{.ProviderVersion}}
Databricks Terraform Provider URL: https://github.com/databricks/terraform-provider-databricks/releases/tag/v{{.ProviderVersion}}

Databricks CLI will download its terraform dependencies automatically.

If you run the CLI in air-gapped environment, you can download the dependencies manually and setup these environment variables:
	DATABRICKS_TF_VERSION={{.Version}}
	DATABRICKS_TF_EXEC_PATH=/path/to/terraform/binary
	DATABRICKS_TF_PROVIDER_VERSION={{.ProviderVersion}}
	DATABRICKS_TF_CLI_CONFIG_FILE=/path/to/terraform/cli/config.tfrc

Example of the *.tfrc config file:
	disable_checkpoint = true
	provider_installation {
		filesystem_mirror {
			path = "/path/to/a/folder/with/databricks/terraform/provider"
		}
	}

The folder with the databricks terraform provider should have this structure: /registry.terraform.io/databricks/databricks/terraform-provider-databricks_{{.ProviderVersion}}_ARCH.zip
Consult terraform docs for more information about the filesystem mirrors: https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror
`,
		},
		// This command is currently intended for the Databricks VSCode extension only
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		dependencies := &Dependencies{
			Terraform: terraform.NewTerraformMetadata(),
		}
		switch root.OutputType(cmd) {
		case flags.OutputText:
			cmdio.Render(cmd.Context(), dependencies.Terraform)
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(dependencies, "", "  ")
			if err != nil {
				return err
			}
			cmd.OutOrStdout().Write(buf)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		return nil
	}

	return cmd
}
