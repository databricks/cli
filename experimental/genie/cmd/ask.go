package geniecmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/genie"
	"github.com/databricks/cli/experimental/genie/agentstream"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	var warehouseID string
	var raw bool
	var includeSQL bool
	var conversationID string

	cmd := &cobra.Command{
		Use:   "ask QUESTION",
		Short: "Ask a data question via Databricks Genie",
		Long: `Ask a data question and get an answer from Databricks Genie.

Examples:
  databricks experimental genie ask "What were total sales last month?"
  databricks experimental genie ask "What tables exist?" --output json
  databricks experimental genie ask "What tables exist?" --warehouse-id abc123
  databricks experimental genie ask "What tables exist?" --raw`,
		Args: root.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// The CLI root doesn't turn Ctrl-C into context cancellation, so scope
			// a SIGINT handler here: an interrupt cancels ctx, which aborts the
			// request (closing the stream signals the server to stop) and lets us
			// exit cleanly instead of dying mid-render.
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
			defer stop()
			outputType := root.OutputType(cmd)
			if raw && outputType == flags.OutputJSON {
				return errors.New("--raw cannot be used with --output json")
			}
			if raw && includeSQL {
				return errors.New("--include-sql cannot be used with --raw")
			}

			w := cmdctx.WorkspaceClient(ctx)
			req := genie.BuildRequest(args[0], warehouseID, conversationID)

			body, err := genie.PostStream(ctx, w.Config, req)
			if err != nil {
				return err
			}
			defer body.Close()

			switch {
			case raw:
				err = agentstream.RenderDebug(body, cmd.OutOrStdout())
			case outputType == flags.OutputJSON:
				err = agentstream.RenderJSON(body, cmd.OutOrStdout(), cmd.ErrOrStderr(), genie.NewAdaptSSE())
			default:
				opts := agentstream.RenderOptions{
					ShowSQL: includeSQL,
					Color:   cmdio.SupportsColor(ctx, cmd.OutOrStdout()),
				}
				err = agentstream.RenderText(ctx, body, cmd.OutOrStdout(), cmd.ErrOrStderr(), genie.NewAdaptSSE(), opts)
			}

			// Ctrl-C: our signal handler cancelled ctx. Exit cleanly rather than
			// dumping "context canceled"; aborting the request already told the
			// server to stop.
			if err != nil && ctx.Err() != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "\nCancelled.")
				return root.ErrAlreadyPrinted
			}
			// The SDK's inactivity timeout cancels the body's read context, so
			// a stalled stream surfaces as context.Canceled while the command's
			// own context is still alive. Translate it; "context canceled" is
			// not actionable.
			if err != nil && errors.Is(err, context.Canceled) && ctx.Err() == nil {
				return fmt.Errorf("the response stream stalled (no data received for %d minutes): %w", genie.StreamingTimeoutSeconds/60, err)
			}
			return err
		},
	}

	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse ID (auto-resolves if omitted)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw SSE events instead of rendered output")
	cmd.Flags().BoolVar(&includeSQL, "include-sql", false, "Show SQL queries executed by the agent (text output; JSON always includes them)")
	cmd.Flags().StringVar(&conversationID, "conversation", "", "Continue an existing conversation by ID (the conversation_id from a prior answer)")

	return cmd
}
