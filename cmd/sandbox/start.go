package sandbox

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
	startPollInterval    = 2 * time.Second
	startWaitTimeout     = 10 * time.Minute
	startRenudgeInterval = 15 * time.Second
)

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <sandbox-id>",
		Short: "Start a stopped Sandbox environment",
		Long: `Start a stopped Sandbox environment.

Boots the backing microVM and blocks until the sandbox reaches
Running (or up to 10 minutes). 'databricks sandbox ssh' already
auto-starts a stopped sandbox on connection, so this command is
mostly useful for pre-warming an environment without immediately
connecting, or when a script needs to be sure the sandbox is up
before continuing.

Starting an already-running sandbox is a no-op.

Example:
  databricks sandbox start happy-panda-1234`,
		Args:              cobra.ExactArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			sandboxID, err := resolveLocalID(ctx, profile, args[0])
			if err != nil {
				return err
			}

			s := spin(ctx, "Starting "+sandboxID+"…")
			defer s.Close()

			updated, err := api.start(ctx, sandboxID)
			if err != nil {
				s.fail("Failed to start " + sandboxID)
				return fmt.Errorf("failed to start sandbox %s: %w", sandboxID, err)
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
				s.fail("Failed to start " + sandboxID)
				return err
			}
			_ = upsertSandbox(ctx, profile, final.SandboxID, final.Name)

			s.ok("Started " + cmdio.Bold(ctx, final.SandboxID))
			return nil
		},
	}

	return cmd
}

type runWaiter interface {
	get(ctx context.Context, id string) (*sandboxEntry, error)
	start(ctx context.Context, id string) (*sandboxEntry, error)
}

func waitForRunning(ctx context.Context, api runWaiter, s *spinner, id string) (*sandboxEntry, error) {
	return waitForState(ctx, api, s, id, "Running", "Starting", "Terminated", "Failed")
}

// The API rejects start requests until teardown finishes, so STOPPING needs
// its own polling phase.
func waitForStopped(ctx context.Context, api runWaiter, s *spinner, id string) (*sandboxEntry, error) {
	return waitForState(ctx, api, s, id, "Stopped", "Stopping", "Terminated", "Failed")
}

// Centralizing lifecycle polling keeps timeout, cancellation, and terminal
// state handling consistent across start and SSH.
func waitForState(ctx context.Context, api runWaiter, s *spinner, id, targetStatus, operation string, unexpectedStatuses ...string) (*sandboxEntry, error) {
	start := time.Now()
	deadline := start.Add(startWaitTimeout)
	lastStart := start
	for {
		sb, err := api.get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("polling status of %s: %w", id, err)
		}
		if strings.EqualFold(sb.Status, targetStatus) {
			return sb, nil
		}
		for _, unexpectedStatus := range unexpectedStatuses {
			if strings.EqualFold(sb.Status, unexpectedStatus) {
				return nil, fmt.Errorf("sandbox %s reached unexpected state %q while %s", id, sb.Status, strings.ToLower(operation))
			}
		}
		elapsed := time.Since(start).Round(time.Second)
		s.Update(fmt.Sprintf("%s %s… (%s)", operation, id, elapsed))
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("sandbox %s did not reach %s within %s (last seen %s)", id, targetStatus, startWaitTimeout, sb.Status)
		}
		if targetStatus == "Running" && strings.EqualFold(sb.Status, "stopped") && time.Since(lastStart) >= startRenudgeInterval {
			_, _ = api.start(ctx, id)
			lastStart = time.Now()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(startPollInterval):
		}
	}
}
