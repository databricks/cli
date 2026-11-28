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

// A resolution that has already finished still counts, with or without the
// wait, even once the budget has passed. The repeats make a race between
// "done" and "not done" fail reliably.
func TestTelemetryAuthFinishedResolutionSurvivesAnExpiredBudget(t *testing.T) {
	origResolve, origBudget := TelemetryAuthResolver, TelemetryAuthBudget
	t.Cleanup(func() { TelemetryAuthResolver, TelemetryAuthBudget = origResolve, origBudget })
	TelemetryAuthResolver = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), &config.Config{Host: "https://ws.test"}))
		return nil
	}
	for _, tc := range []struct {
		wait   bool
		budget time.Duration
	}{{wait: true, budget: 0}, {wait: false, budget: time.Second}} {
		TelemetryAuthBudget = tc.budget
		for range 50 {
			cmd := &cobra.Command{Use: "test"}
			cmd.SetContext(env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", ""))
			p := startTelemetryAuth(cmd, nil)
			<-p.done
			telemetry.Log(cmd.Context(), protos.DatabricksCliLog{AitoolsInstallEvent: &protos.AitoolsInstallEvent{}})
			p.apply(cmd, tc.wait)
			require.True(t, cmdctx.HasConfigUsed(cmd.Context()))
		}
	}
}
