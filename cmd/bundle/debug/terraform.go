package debug

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

func NewTerraformCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terraform",
		Short: "Prints Terraform dependencies required for the bundle commands",
		Args:  root.NoArgs,
		Annotations: map[string]string{
			"template": `Terraform version: {{.Version}}
Terraform URL: https://releases.hashicorp.com/terraform/{{.Version}}

Databricks Terraform Provider version: {{.ProviderVersion}}
Databricks Terraform Provider URL: https://github.com/databricks/terraform-provider-databricks/releases/tag/v{{.ProviderVersion}}

Databricks CLI downloads its Terraform dependencies automatically.

If you run the CLI in an air-gapped environment, you can download the dependencies manually and set these environment variables:

	DATABRICKS_TF_VERSION={{.Version}}
	DATABRICKS_TF_EXEC_PATH=/path/to/terraform/binary
	DATABRICKS_TF_PROVIDER_VERSION={{.ProviderVersion}}
	DATABRICKS_TF_CLI_CONFIG_FILE=/path/to/terraform/cli/config.tfrc

Here is an example *.tfrc configuration file:

	disable_checkpoint = true
	provider_installation {
		filesystem_mirror {
			path = "/path/to/a/folder/with/databricks/terraform/provider"
		}
	}

The filesystem mirror path should point to the folder with the Databricks Terraform Provider. The folder should have this structure: /{{.ProviderHost}}/{{.ProviderSource}}/terraform-provider-databricks_{{.ProviderVersion}}_ARCH.zip

For more information about filesystem mirrors, see the Terraform documentation: https://developer.hashicorp.com/terraform/cli/config/config-file#filesystem_mirror
`,
		},
		// This command is currently intended for the Databricks VSCode extension only
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		tv, err := terraform.NewTerraformMetadata(cmd.Context())
		if err != nil {
			return err
		}
		dependencies := &Dependencies{
			Terraform: tv,
		}
		switch root.OutputType(cmd) {
		case flags.OutputText:
			_ = cmdio.Render(cmd.Context(), dependencies.Terraform)
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(dependencies, "", "  ")
			if err != nil {
				return err
			}
			_, _ = cmd.OutOrStdout().Write(buf)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}

		return nil
	}

	return cmd
}
