package debug

import (
	"fmt"
	"io"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func NewSchemaMapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema-map",
		Short: "Print the DABs↔Terraform field mapping summary",
		Long: `Print a per-resource summary of the bidirectional field mapping between
DABs resource fields and Terraform resource fields.

Each output line is:

  resource (tf_type): N renames, N dabs-only, N tf-only

Status meanings:
  renames   fields that exist in both under different leaf names
  dabs-only fields present in DABs but not in Terraform
  tf-only   fields present in Terraform but not in DABs
`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return printSchemaMap(cmd.OutOrStdout())
	}

	return cmd
}

func printSchemaMap(out io.Writer) error {
	for _, group := range terraform_dabs_map.Groups {
		tfType := terraform.GroupToTerraformName[group]
		renames := len(terraform_dabs_map.TerraformToDABsFieldMap[group])
		dabsOnly := len(terraform_dabs_map.DABsOnlyFields[group])
		tfOnly := len(terraform_dabs_map.TerraformOnlyFields[group])
		fmt.Fprintf(out, "%s (%s): %d renames, %d dabs-only, %d tf-only\n",
			group, tfType, renames, dabsOnly, tfOnly)
	}
	return nil
}
