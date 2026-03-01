package onechat

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/agentstream"
	onechatlib "github.com/databricks/cli/experimental/onechat"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	var warehouseID string
	var debug bool

	cmd := &cobra.Command{
		Use:   "ask QUESTION",
		Short: "Ask a data question via Databricks One Chat",
		Long: `Ask a data question and get an answer from Databricks One Chat.

Examples:
  databricks experimental onechat ask "What were total sales last month?"
  databricks experimental onechat ask "What tables exist?" --output json
  databricks experimental onechat ask "What tables exist?" --warehouse-id abc123
  databricks experimental onechat ask "What tables exist?" --debug`,
		Args: root.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			question := args[0]
			req := onechatlib.BuildRequest(question, warehouseID)

			body, err := onechatlib.PostStream(ctx, w.Config, req)
			if err != nil {
				return err
			}
			defer body.Close()

			if debug {
				return agentstream.RenderDebug(body, cmd.OutOrStdout())
			}

			outputType := root.OutputType(cmd)
			if outputType == flags.OutputJSON {
				return agentstream.RenderJSON(body, cmd.OutOrStdout(), onechatlib.AdaptSSE)
			}

			return agentstream.RenderText(body, cmd.OutOrStdout(), cmd.ErrOrStderr(), onechatlib.AdaptSSE)
		},
	}

	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse ID (auto-resolves if omitted)")
	cmd.Flags().BoolVar(&debug, "debug", false, "Print raw SSE events for debugging")

	return cmd
}
