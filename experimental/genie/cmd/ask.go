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
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	var warehouseID string
	var raw bool
	var includeSQL bool
	var conversation string

	cmd := &cobra.Command{
		Use:   "ask QUESTION",
		Short: "Ask a data question via Databricks Genie",
		Long: `Ask a data question and get an answer from Databricks Genie.

Examples:
  databricks experimental genie ask "What were total sales last month?"
  databricks experimental genie ask "What tables exist?" --output json
  databricks experimental genie ask "Break that down by region" --conversation q3
  databricks experimental genie ask "What tables exist?" --raw`,
		Args: root.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// The CLI root doesn't turn Ctrl-C into context cancellation, so scope
			// a SIGINT handler here: an interrupt cancels ctx, aborting the stream.
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
			host := w.Config.Host

			// Only touch the on-disk cache when a label is in play.
			var convCache *cache.Cache
			if conversation != "" {
				convCache = newConversationCache(ctx)
			}

			// ask runs one request+render against serverID ("" starts a fresh
			// conversation) and reports the conversation id from the response,
			// whether anything reached stdout, and any error.
			ask := func(serverID string) (id string, wrote bool, err error) {
				body, err := genie.PostStream(ctx, w.Config, genie.BuildRequest(args[0], warehouseID, serverID))
				if err != nil {
					return "", false, err
				}
				defer body.Close()
				switch {
				case raw:
					return "", true, agentstream.RenderDebug(body, cmd.OutOrStdout())
				case outputType == flags.OutputJSON:
					id, err := agentstream.RenderJSON(body, cmd.OutOrStdout(), cmd.ErrOrStderr(), genie.NewAdaptSSE())
					return id, true, err
				default:
					opts := agentstream.RenderOptions{ShowSQL: includeSQL, Color: cmdio.SupportsColor(ctx, cmd.OutOrStdout())}
					return agentstream.RenderText(ctx, body, cmd.OutOrStdout(), cmd.ErrOrStderr(), genie.NewAdaptSSE(), opts)
				}
			}

			serverID := lookupConversationID(ctx, convCache, host, conversation)
			id, wrote, err := ask(serverID)

			// Fail open: resuming an expired/unknown conversation errors before any
			// output (and is not a cancel/stall, which surface as context.Canceled).
			// Forget the dead mapping, and if nothing was written yet, retry as a
			// fresh conversation so the label keeps working.
			if err != nil && serverID != "" && !errors.Is(err, context.Canceled) {
				forgetConversation(ctx, convCache, host, conversation)
				if !wrote {
					fmt.Fprintf(cmd.ErrOrStderr(), "Conversation %q was not found (it may have expired); starting a new one.\n", conversation)
					id, wrote, err = ask("")
				}
			}

			switch {
			case err != nil && ctx.Err() != nil:
				// Ctrl-C: aborting the request already told the server to stop.
				fmt.Fprintln(cmd.ErrOrStderr(), "\nCancelled.")
				return root.ErrAlreadyPrinted
			case err != nil && errors.Is(err, context.Canceled):
				// The SDK's inactivity timeout cancelled the body read while our
				// own context is still alive: the stream stalled.
				return fmt.Errorf("the response stream stalled (no data received for %d minutes): %w", genie.StreamingTimeoutSeconds/60, err)
			case err != nil:
				return err
			}

			storeConversationID(ctx, convCache, host, conversation, id)
			return nil
		},
	}

	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse ID (auto-resolves if omitted)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw SSE events instead of rendered output")
	cmd.Flags().BoolVar(&includeSQL, "include-sql", false, "Show SQL queries executed by the agent (text output; JSON always includes them)")
	cmd.Flags().StringVarP(&conversation, "conversation", "c", "", "Conversation label (any string) to continue across calls")

	return cmd
}
