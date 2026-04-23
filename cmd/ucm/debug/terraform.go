package debug

import (
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	ucmtf "github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/spf13/cobra"
)

// ProviderHost is the registry host the databricks provider publishes under.
// Forked from bundle/internal/tf/schema.ProviderHost to honour the ucm
// no-bundle-imports rule; kept verbatim so air-gap mirrors share a layout.
const ProviderHost = "registry.terraform.io"

// TerraformMetadata is the JSON payload returned by `ucm debug terraform -o json`.
// Shape forked from bundle/deploy/terraform.TerraformMetadata, trimmed to the
// fields ucm actually pins (checksums are bundle-only). Tag names match bundle's
// so downstream tools can consume both outputs with one parser.
type TerraformMetadata struct {
	Version         string `json:"version"`
	ProviderHost    string `json:"providerHost"`
	ProviderSource  string `json:"providerSource"`
	ProviderVersion string `json:"providerVersion"`
}

// Dependencies wraps TerraformMetadata to match cmd/bundle/debug's JSON shape.
type Dependencies struct {
	Terraform *TerraformMetadata `json:"terraform"`
}

// terraformTemplate is the text rendered for `ucm debug terraform`. Copied
// verbatim from cmd/bundle/debug/terraform.go with the binary name swapped
// so air-gap instructions read correctly for the ucm subcommand.
const terraformTemplate = `Terraform version: {{.Version}}
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
`

// NewTerraformCommand returns the `ucm debug terraform` command. Prints the
// terraform CLI and databricks-provider versions ucm pins, plus the env vars
// users can set to run in an air-gapped environment.
func NewTerraformCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terraform",
		Short: "Prints Terraform dependencies required for the ucm commands",
		Args:  root.NoArgs,
		Annotations: map[string]string{
			// Kept for parity with cmd/bundle/debug/terraform.go so VSCode
			// tooling that reads the annotation sees the same template.
			"template": terraformTemplate,
		},
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		tv, _, err := ucmtf.GetTerraformVersion(cmd.Context())
		if err != nil {
			return err
		}
		deps := &Dependencies{
			Terraform: &TerraformMetadata{
				Version:         tv.String(),
				ProviderHost:    ProviderHost,
				ProviderSource:  ucmtf.ProviderSource,
				ProviderVersion: ucmtf.ProviderVersion,
			},
		}

		switch outputType(cmd) {
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(deps, "", "  ")
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(buf)
			return err
		case flags.OutputText:
			t, err := template.New("terraform").Parse(terraformTemplate)
			if err != nil {
				return err
			}
			return t.Execute(cmd.OutOrStdout(), deps.Terraform)
		default:
			return fmt.Errorf("unknown output type %s", outputType(cmd))
		}
	}

	return cmd
}

// outputType returns the configured -o value, defaulting to OutputText when
// the flag is not wired (e.g. unit tests that don't go through root.New).
// Mirrors plan.go's planOutputType — root.OutputType would panic in that case.
func outputType(cmd *cobra.Command) flags.Output {
	if cmd.Flag("output") == nil {
		return flags.OutputText
	}
	return root.OutputType(cmd)
}
