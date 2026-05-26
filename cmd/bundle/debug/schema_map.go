package debug

import (
	"fmt"
	"io"

	"github.com/databricks/cli/bundle/terraform_dabs_map"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func NewSchemaMapCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema-map",
		Short: "Print the DABs↔Terraform field mapping",
		Long: `Print the bidirectional field mapping between DABs resource fields and
Terraform resource fields. Each output line is tab-separated:

  status  dabs_path  tf_resource  tf_path

Status values:
  match      field exists in both DABs and Terraform with the same name
  renamed    field exists in both but under different names (e.g. tasks→task)
  dabs_only  field exists in DABs but not in Terraform
  tf_only    field exists in Terraform but not in DABs
  no_tf_type DABs resource has no known Terraform resource type
`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return printSchemaMap(cmd.OutOrStdout())
	}

	return cmd
}

func printSchemaMap(out io.Writer) error {
	entries, err := terraform_dabs_map.Build()
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "status\tdabs_path\ttf_resource\ttf_path")
	for _, e := range entries {
		fmt.Fprintf(out, "%s\t%s\t%s\t%s\n", e.Status, e.DabsPath, e.TFType, e.TFPath)
	}
	return nil
}
