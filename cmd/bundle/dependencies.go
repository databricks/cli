package bundle

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	Terraform *terraform.TerraformMetadata `json:"terraform"`
}

func newDependenciesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dependencies",
		Short: "Prints CLI dependencies required for the bundle namespace",
		Args:  root.NoArgs,
		// This command is currently intended for the Databricks VSCode extension only
		Hidden: true,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		dependencies := &Dependencies{
			Terraform: terraform.NewTerraformMetadata(),
		}
		switch root.OutputType(cmd) {
		case flags.OutputText:
			return fmt.Errorf("%w, only json output is supported", errors.ErrUnsupported)
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
