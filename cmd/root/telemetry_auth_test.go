package root

import (
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// setTelemetryAuth replaces the auth resolver and the wait budget for a test.
func setTelemetryAuth(t *testing.T, budget time.Duration, resolve func(cmd *cobra.Command, args []string) error) {
	origResolve, origBudget := TelemetryAuthResolver, TelemetryAuthBudget
	t.Cleanup(func() { TelemetryAuthResolver, TelemetryAuthBudget = origResolve, origBudget })
	TelemetryAuthResolver = resolve
	TelemetryAuthBudget = budget
}

// authTestCmd returns a command wired with WithTelemetryAuth whose RunE logs a
// telemetry event, plus a --profile flag like root's. Telemetry is enabled on
// its context whatever the outer environment says.
func authTestCmd(t *testing.T) (*cobra.Command, *bool) {
	ran := false
	cmd := &cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			ran = true
			telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
			return nil
		},
	}
	cmd.Flags().String("profile", "", "")
	WithTelemetryAuth(cmd).WaitAtExit = true
	ctx := telemetry.WithNewLogger(t.Context())
	cmd.SetContext(env.Set(ctx, "DATABRICKS_CLI_DISABLE_TELEMETRY", ""))
	return cmd, &ran
}

// A command that outlasts the budget still uploads when its resolution
// finished in time. Budget 0 makes every wait start past the deadline; the
// repeats make a coin-flip between "done" and "expired" fail reliably.
func TestTelemetryAuthFinishedResolutionSurvivesAnExpiredBudget(t *testing.T) {
	setTelemetryAuth(t, 0, func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	})
	for range 50 {
		cmd, _ := authTestCmd(t)
		p := startTelemetryAuth(cmd, nil)
		<-p.done
		telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
		p.apply(cmd, true)
		require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
	}
}

// Without the wait, a resolution that has already finished still counts. The
// repeats make a race between "done" and "not done" fail reliably.
func TestTelemetryAuthNoWaitUsesFinishedResolution(t *testing.T) {
	setTelemetryAuth(t, time.Second, func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	})
	for range 50 {
		cmd, _ := authTestCmd(t)
		p := startTelemetryAuth(cmd, nil)
		<-p.done
		telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
		p.apply(cmd, false)
		require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
	}
}
