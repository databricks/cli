package debug

import (
	"fmt"
	"io"
	"maps"
	"slices"

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
	// Collect the union of all groups that appear in any generated map.
	// This naturally excludes 3-level groups (permissions, grants, secret_acls)
	// which have no adapter and therefore no entries in any map.
	groups := make(map[string]bool)
	for g := range terraform_dabs_map.TerraformToDABsFieldMap {
		groups[g] = true
	}
	for g := range terraform_dabs_map.DABsOnlyFields {
		groups[g] = true
	}
	for g := range terraform_dabs_map.TerraformOnlyFields {
		groups[g] = true
	}
	for g := range terraform_dabs_map.TerraformComputedFields {
		groups[g] = true
	}

	for _, group := range slices.Sorted(maps.Keys(groups)) {
		tfType := terraform.GroupToTerraformName[group]
		renames := terraform_dabs_map.TerraformToDABsFieldMap[group].Len()
		dabsOnly := terraform_dabs_map.DABsOnlyFields[group].Len()
		tfOnly := terraform_dabs_map.TerraformOnlyFields[group].Len()
		computed := terraform_dabs_map.TerraformComputedFields[group].Len()
		fmt.Fprintf(out, "%s (%s): %d renames, %d dabs-only, %d tf-only, %d computed\n",
			group, tfType, renames, dabsOnly, tfOnly, computed)
	}
	return nil
}
