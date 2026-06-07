package lakebox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Bounds for `start`'s "wait until Running" poll. StartSandbox returns
// immediately with status="Creating", so we poll until it actually
// reaches Running. 10 min covers the observed cold-start range
// (5–13 min); stuck sandboxes surface as a timeout, not a hang.
const (
	startPollInterval = 2 * time.Second
	startWaitTimeout  = 10 * time.Minute
)

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <lakebox-id>",
		Short: "Start a stopped Lakebox environment",
		Long: `Start a stopped Lakebox environment.

Boots the backing microVM and blocks until the sandbox reaches
Running (or up to 10 minutes). 'databricks lakebox ssh' already
auto-starts a stopped sandbox on connection, so this command is
mostly useful for pre-warming an environment without immediately
connecting, or when a script needs to be sure the sandbox is up
before continuing.

Starting an already-running sandbox is a no-op.

Example:
  databricks lakebox start happy-panda-1234`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			lakeboxID, err := resolveLocalID(ctx, profile, args[0])
			if err != nil {
				return err
			}

			s := spin(ctx, "Starting "+lakeboxID+"…")
			defer s.Close()

			updated, err := api.start(ctx, lakeboxID)
			if err != nil {
				s.fail("Failed to start " + lakeboxID)
				return fmt.Errorf("failed to start lakebox %s: %w", lakeboxID, err)
			}

			_ = setGatewayHost(ctx, profile, updated.GatewayHost)
			_ = upsertSandbox(ctx, profile, updated.SandboxID, updated.Name)

			// Already up — short-circuit so we don't pretend to wait
			// when there's nothing to wait for.
			if strings.EqualFold(updated.Status, "running") {
				s.ok("Already running " + cmdio.Bold(ctx, updated.SandboxID))
				return nil
			}

			final, err := waitForRunning(ctx, api, s, updated.SandboxID)
			if err != nil {
				s.fail("Failed to start " + lakeboxID)
				return err
			}
			_ = upsertSandbox(ctx, profile, final.SandboxID, final.Name)

			s.ok("Started " + cmdio.Bold(ctx, final.SandboxID))
			return nil
		},
	}

	return cmd
}

// waitForRunning polls the sandbox until it reaches Running or the timeout
// elapses. The spinner is updated with the elapsed seconds each poll so the
// user can tell progress is happening, and a transition to an unexpected
// terminal state (Stopped / Terminated / Failed) short-circuits with a
// useful error rather than waiting out the full timeout.
func waitForRunning(ctx context.Context, api *lakeboxAPI, s *spinner, id string) (*sandboxEntry, error) {
	start := time.Now()
	deadline := start.Add(startWaitTimeout)
	for {
		sb, err := api.get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("polling status of %s: %w", id, err)
		}
		switch strings.ToLower(sb.Status) {
		case "running":
			return sb, nil
		case "stopped", "terminated", "failed":
			return nil, fmt.Errorf("sandbox %s reached unexpected state %q while starting", id, sb.Status)
		}
		elapsed := time.Since(start).Round(time.Second)
		s.Update(fmt.Sprintf("Starting %s… (%s)", id, elapsed))
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("sandbox %s did not reach Running within %s (last seen %s)", id, startWaitTimeout, sb.Status)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(startPollInterval):
		}
	}
}
