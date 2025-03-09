package bundle

import (
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/clis"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newDryRunCommand(hidden bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dry-run [flags] KEY",
		Short: "Start a dry run",
		Long: `Start a dry run of the DLT pipeline identified by KEY.
This command is a short-hand for 'databricks bundle run --validate-only KEY

The KEY is the unique identifier of the resource to run, for example:

   databricks bundle dry-run my_dlt
`,
		Hidden: hidden,
	}
	runCmd := newRunCommand(clis.DLT)

	var pipelineOpts run.PipelineOptions
	pipelineOpts.Define(cmd.Flags())

	// Reuse the run command's implementation but with our pipeline options
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		err := runCmd.Flags().Set("validate-only", "true")
		if err != nil {
			return err
		}

		err = runCmd.RunE(cmd, nil)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, "✅ dry run successful, no problems found!")
		return nil
	}
	cmd.ValidArgsFunction = runCmd.ValidArgsFunction

	return cmd
}
