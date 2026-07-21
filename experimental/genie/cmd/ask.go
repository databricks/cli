package geniecmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

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
	var session string

	cmd := &cobra.Command{
		Use:   "ask QUESTION",
		Short: "Ask a data question via Databricks Genie",
		Long: `Ask a data question and get an answer from Databricks Genie.

Examples:
  databricks experimental genie ask "What were total sales last month?"
  databricks experimental genie ask "What tables exist?" --output json
  databricks experimental genie ask "Revenue by region" --warehouse-id 1234567890abcdef
  databricks experimental genie ask "What tables exist?" --raw

  # Continue a conversation across calls with a session id you choose:
  databricks experimental genie ask -s sales "What were total sales by quarter?"
  databricks experimental genie ask -s sales "Break that down by region"`,
		Args: root.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// The CLI root doesn't cancel ctx on signals, so handle them here:
			// SIGINT (Ctrl-C) and SIGTERM (kill — how an agent stops a child)
			// cancel ctx, aborting the stream.
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			outputType := root.OutputType(cmd)
			if raw && outputType == flags.OutputJSON {
				return errors.New("--raw cannot be used with --output json")
			}
			if raw && includeSQL {
				return errors.New("--include-sql cannot be used with --raw")
			}
			if raw && session != "" {
				// --raw doesn't parse a conversation id, so it can't be tracked.
				fmt.Fprintln(cmd.ErrOrStderr(), "Warning: --session has no effect with --raw (the conversation is not tracked).")
			}

			w := cmdctx.WorkspaceClient(ctx)
			host := w.Config.Host

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

			err := askWithConversation(ctx, cmd.ErrOrStderr(), host, session, ask)
			switch {
			case err != nil && ctx.Err() != nil:
				// Interrupted (Ctrl-C / kill): aborting the request already told
				// the server to stop.
				fmt.Fprintln(cmd.ErrOrStderr(), "\nCancelled.")
				return root.ErrAlreadyPrinted
			case err != nil && errors.Is(err, context.Canceled):
				// The SDK's inactivity timeout cancelled the body read while our
				// own context is still alive: the stream stalled.
				return fmt.Errorf("the response stream stalled (no data received for %d minutes): %w", genie.StreamingTimeoutSeconds/60, err)
			default:
				return err
			}
		},
	}

	cmd.Flags().StringVar(&warehouseID, "warehouse-id", "", "SQL warehouse ID (auto-resolves if omitted)")
	cmd.Flags().BoolVar(&raw, "raw", false, "Print raw SSE events instead of rendered output")
	cmd.Flags().BoolVar(&includeSQL, "include-sql", false, "Show SQL queries executed by the agent (text output; JSON always includes them)")
	cmd.Flags().StringVarP(&session, "session", "s", "", "Session id (any string) to continue a conversation across calls")

	return cmd
}

// askWithConversation resolves sessionID to its stored conversation id, runs
// ask, and keeps the session id usable. A resume failure (not a cancel/stall,
// which surface as context.Canceled) forgets the dead mapping and, if nothing
// was written yet, retries as a fresh conversation. On success the resulting
// conversation id is stored. An empty sessionID disables all of this and just
// runs ask once.
func askWithConversation(ctx context.Context, stderr io.Writer, host, sessionID string, ask func(serverID string) (string, bool, error)) error {
	serverID := lookupConversationID(ctx, host, sessionID)
	id, wrote, err := ask(serverID)
	if err != nil && serverID != "" && !errors.Is(err, context.Canceled) {
		forgetConversation(ctx, host, sessionID)
		if !wrote {
			fmt.Fprintf(stderr, "Session %q was not found (it may have expired); starting a new one.\n", sessionID)
			id, _, err = ask("")
		}
	}
	if err != nil {
		return err
	}
	storeConversationID(ctx, host, sessionID, id)
	return nil
}
