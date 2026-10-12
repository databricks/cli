package debug

import (
	"context"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

func NewStatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "states",
		Short: "Show available state files",
		Args:  root.NoArgs,
	}

	var forcePull bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			SkipInitialize: !forcePull,
			ReadState:      true,
			AlwaysPull:     forcePull,
			PostInitFunc: func(ctx context.Context, b *bundle.Bundle) error {
				if forcePull {
					return nil
				}
				var configured *bool
				if b.Config.Experimental != nil {
					configured = b.Config.Experimental.DeploymentHistory
				}
				if !bundleenv.RecordsDeploymentHistory(ctx, configured) {
					return nil
				}
				// Deployment history reads remote state even without --force-pull.
				t0 := time.Now()
				phases.Initialize(ctx, b)
				b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
					Key:   "phases.Initialize",
					Value: time.Since(t0).Milliseconds(),
				})
				if logdiag.HasError(ctx) {
					return root.ErrAlreadyPrinted
				}
				return nil
			},
		}

		_, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		ctx := cmd.Context()

		if err != nil {
			return err
		}

		var stateStrs []string
		for _, state := range stateDesc.AllStates {
			stateStrs = append(stateStrs, state.String())
		}
		cmdio.LogString(ctx, strings.Join(stateStrs, "\n"))
		return nil
	}

	return cmd
}
