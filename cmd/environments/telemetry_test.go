package environments

import (
	"testing"

	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
)

func TestBuildSetupLocalEvent(t *testing.T) {
	t.Run("dry-run emits no event", func(t *testing.T) {
		res := &libslocalenv.Result{DryRun: true, OK: true}
		assert.Nil(t, buildSetupLocalEvent(res))
	})

	t.Run("nil result emits no event", func(t *testing.T) {
		assert.Nil(t, buildSetupLocalEvent(nil))
	})

	t.Run("success run maps fields", func(t *testing.T) {
		res := &libslocalenv.Result{
			OK:         true,
			Mode:       libslocalenv.ModeDefault.String(),
			Greenfield: true,
			Compute:    &libslocalenv.ComputeInfo{Source: "serverless", EnvKey: "serverless/serverless-v5"},
		}
		got := buildSetupLocalEvent(res)
		assert.Equal(t, &protos.SetupLocalEvent{
			Success:       true,
			ComputeSource: protos.SetupLocalComputeSourceServerless,
			Mode:          protos.SetupLocalModeDefault,
			Greenfield:    true,
			EnvKey:        "serverless/serverless-v5",
		}, got)
	})

	t.Run("failure run maps error code, phase, diskMutated", func(t *testing.T) {
		res := &libslocalenv.Result{
			OK:      false,
			Mode:    libslocalenv.ModeConstraintsOnly.String(),
			Compute: &libslocalenv.ComputeInfo{Source: "cluster", EnvKey: "dbr/16.4"},
			Error: &libslocalenv.PipelineError{
				Code:         libslocalenv.ErrProvision,
				FailurePhase: libslocalenv.PhaseProvision,
				DiskMutated:  true,
			},
		}
		got := buildSetupLocalEvent(res)
		assert.Equal(t, &protos.SetupLocalEvent{
			Success:       false,
			ComputeSource: protos.SetupLocalComputeSourceCluster,
			Mode:          protos.SetupLocalModeConstraintsOnly,
			EnvKey:        "dbr/16.4",
			ErrorCode:     protos.SetupLocalErrorCodeProvision,
			FailurePhase:  protos.SetupLocalPhaseProvision,
			DiskMutated:   true,
		}, got)
	})

	t.Run("nil compute leaves source and envKey empty", func(t *testing.T) {
		res := &libslocalenv.Result{
			OK:    false,
			Mode:  libslocalenv.ModeDefault.String(),
			Error: &libslocalenv.PipelineError{Code: libslocalenv.ErrNoTarget, FailurePhase: libslocalenv.PhaseResolve},
		}
		got := buildSetupLocalEvent(res)
		assert.Equal(t, protos.SetupLocalComputeSourceUnspecified, got.ComputeSource)
		assert.Empty(t, got.EnvKey)
		assert.Equal(t, protos.SetupLocalErrorCodeNoTarget, got.ErrorCode)
	})
}
