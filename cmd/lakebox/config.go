package lakebox

import (
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// MIN_IDLE_TIMEOUT_SECS / MAX_IDLE_TIMEOUT_SECS mirror the manager-side
// constants in lakebox/src/api/handlers/sandbox.rs. Pre-flighting client-side
// gives a clearer error than waiting for the server's INVALID_ARGUMENT.
const (
	minIdleTimeoutSecs = 60
	maxIdleTimeoutSecs = 86_400
)

func newConfigCommand() *cobra.Command {
	var idleTimeoutFlag string
	var noAutostopFlag bool
	var nameFlag string

	cmd := &cobra.Command{
		Use:   "config <lakebox-id>",
		Short: "Configure a Lakebox's name and auto-stop policy",
		Long: `Configure a Lakebox's name and auto-stop policy.

Three knobs are independent — pass any combination:

  --name <label>              Display label for the lakebox (max 256 bytes).
                              Pass an empty string to clear.

  --idle-timeout <duration>   Per-sandbox idle timeout. The watchdog reaps
                              the sandbox after this much idle time. Pass
                              0 (or 0s) to clear and revert to the manager's
                              global default (10m). Valid range when set:
                              60s to 24h.

  --no-autostop[=true|false]  When true, the sandbox is exempt from
                              idle-driven auto-stop entirely. The
                              --idle-timeout setting is ignored while
                              this is on. Setting --idle-timeout to a
                              non-zero value in a later call clears
                              --no-autostop automatically. Sandbox still
                              stops on explicit 'databricks lakebox delete'.

Examples:
  databricks lakebox config happy-panda-1234 --name my-project
  databricks lakebox config happy-panda-1234 --idle-timeout 15m
  databricks lakebox config happy-panda-1234 --idle-timeout 1h30m
  databricks lakebox config happy-panda-1234 --idle-timeout 0           # clear, use default
  databricks lakebox config happy-panda-1234 --no-autostop                  # never auto-stop
  databricks lakebox config happy-panda-1234 --no-autostop=false            # back to timeout path
  databricks lakebox config happy-panda-1234 --idle-timeout 30m --no-autostop=false`,
		Args:    cobra.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)
			api, err := newLakeboxAPI(w)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()

			id := args[0]

			// Translate flag presence + value into the proto3
			// optional-field semantics the server expects.
			var idleSecs *int64
			if cmd.Flags().Changed("idle-timeout") {
				secs, err := parseIdleTimeoutFlag(idleTimeoutFlag)
				if err != nil {
					return err
				}
				idleSecs = &secs
			}

			var noAutostop *bool
			if cmd.Flags().Changed("no-autostop") {
				p := noAutostopFlag
				noAutostop = &p
			}

			var name *string
			if cmd.Flags().Changed("name") {
				n := nameFlag
				name = &n
			}

			if idleSecs == nil && noAutostop == nil && name == nil {
				return errors.New("nothing to update — pass --name, --idle-timeout, and/or --no-autostop")
			}

			updated, err := api.update(ctx, id, name, idleSecs, noAutostop)
			if err != nil {
				return fmt.Errorf("failed to update lakebox %s: %w", id, err)
			}

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}
			_ = setGatewayHost(ctx, profile, updated.GatewayHost)

			blank(out)
			field(ctx, out, "id", cmdio.Bold(ctx, updated.SandboxID))
			if updated.Name != "" {
				field(ctx, out, "name", updated.Name)
			}
			field(ctx, out, "autostop", cmdio.Faint(ctx, updated.autoStopLabel()))
			blank(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&nameFlag, "name", "",
		"Display label for the lakebox (max 256 bytes). Pass --name= to clear.")
	cmd.Flags().StringVar(&idleTimeoutFlag, "idle-timeout", "",
		"Idle timeout (e.g. 15m, 1h30m, 90s). Pass 0 to clear and revert to the manager's default.")
	cmd.Flags().BoolVar(&noAutostopFlag, "no-autostop", false,
		"When true, this sandbox never auto-stops on idle. Pass --no-autostop=false to revert.")

	return cmd
}

// parseIdleTimeoutFlag accepts the same syntax as time.ParseDuration plus
// the special-case "0" / "0s" → clear. Anything else outside the
// [60s, 86400s] window is rejected client-side.
func parseIdleTimeoutFlag(raw string) (int64, error) {
	d, err := time.ParseDuration(raw)
	if err != nil {
		// Allow bare integer seconds as a convenience (`--idle-timeout 900`).
		var secs int64
		if _, e2 := fmt.Sscanf(raw, "%d", &secs); e2 == nil {
			return checkIdleSecs(secs)
		}
		return 0, fmt.Errorf("invalid --idle-timeout %q: %w (use Go duration syntax, e.g. 15m, 1h30m)", raw, err)
	}
	return checkIdleSecs(int64(d.Seconds()))
}

func checkIdleSecs(secs int64) (int64, error) {
	if secs == 0 {
		return 0, nil // clear / revert to global default
	}
	if secs < minIdleTimeoutSecs || secs > maxIdleTimeoutSecs {
		return 0, fmt.Errorf(
			"idle-timeout must be 0 (clear) or between %ds and %ds, got %ds",
			minIdleTimeoutSecs, maxIdleTimeoutSecs, secs,
		)
	}
	return secs, nil
}
