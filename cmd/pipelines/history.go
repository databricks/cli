package pipelines

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func historyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history [flags] PIPELINE_ID",
		Short: "Retrieve past runs for a pipeline",
		Long:  `Retrieve past runs for a pipeline identified by PIPELINE_ID, a unique identifier for a pipeline.`,
	}

	var maxResults int

	historyGroup := cmdgroup.NewFlagGroup("Filter")
	historyGroup.FlagSet().IntVar(&maxResults, "max-results", 100, "Max number of entries in output.")
	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(historyGroup)

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 0 {
			return errors.New("Provide a PIPELINE_ID.")
		}

		if len(args) > 1 {
			return fmt.Errorf("Expected one PIPELINE_ID, got %d.", len(args))
		}
		w := cmdctx.WorkspaceClient(ctx)

		pipelineId := args[0]

		req := pipelines.ListUpdatesRequest{
			PipelineId: pipelineId,
			MaxResults: maxResults,
		}

		response, err := w.Pipelines.ListUpdates(ctx, req)
		if err != nil {
			return err
		}

		return cmdio.RenderWithTemplate(ctx, response, fmt.Sprintf("Updates summary for pipeline %s", pipelineId),
			`{{range .Updates}}Update ID: {{.UpdateId}}
     State: {{.State}}
     Cause: {{.Cause}}
     Creation Time: {{.CreationTime}}
     Full Refresh: {{.FullRefresh}}
     Validate Only: {{.ValidateOnly}}
	 
{{end}}`)
	}

	return cmd
}
