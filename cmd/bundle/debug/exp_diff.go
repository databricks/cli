package debug

import (
	"encoding/json"

	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func NewExpDiffCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exp-diff",
		Short: "Show resolved bundle configuration in JSON format (experimental)",
		Long:  "Show the resolved bundle configuration after all variable substitutions and includes have been processed. This command is experimental and may change without notice.",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{})
		if err != nil {
			return err
		}

		buf, err := json.MarshalIndent(b.Config.Value().AsAny(), "", "  ")
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		_, _ = out.Write(buf)
		_, _ = out.Write([]byte{'\n'})

		return nil
	}

	return cmd
}
