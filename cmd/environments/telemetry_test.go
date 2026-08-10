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

// TestErrorCodeCoversLocalenv fails when a new localenv.ErrorCode is added
// without a matching telemetry enum in errorCodeType. Keep this list in sync
// with the const block in libs/localenv/result.go.
func TestErrorCodeCoversLocalenv(t *testing.T) {
	all := []libslocalenv.ErrorCode{
		libslocalenv.ErrUsage,
		libslocalenv.ErrManagerUnsupported,
		libslocalenv.ErrNotWritable,
		libslocalenv.ErrUvMissing,
		libslocalenv.ErrNoTarget,
		libslocalenv.ErrResolve,
		libslocalenv.ErrEnvUnsupported,
		libslocalenv.ErrFetch,
		libslocalenv.ErrWrite,
		libslocalenv.ErrMerge,
		libslocalenv.ErrPythonInstall,
		libslocalenv.ErrProvision,
		libslocalenv.ErrValidate,
		libslocalenv.ErrCanceled,
	}
	for _, code := range all {
		assert.NotEqualf(t, protos.SetupLocalErrorCodeUnspecified, errorCodeType(code),
			"error code %q has no telemetry enum: add a case to errorCodeType and a "+
				"value to SetupLocalErrorCode (setup_local.go + universe proto)", code)
	}
}

// TestPhaseCoversLocalenv fails when a new localenv.PhaseName is added without a
// matching telemetry enum in phaseType.
func TestPhaseCoversLocalenv(t *testing.T) {
	all := []libslocalenv.PhaseName{
		libslocalenv.PhasePreflight,
		libslocalenv.PhaseResolve,
		libslocalenv.PhaseFetch,
		libslocalenv.PhaseMerge,
		libslocalenv.PhaseProvision,
		libslocalenv.PhaseValidate,
	}
	for _, phase := range all {
		assert.NotEqualf(t, protos.SetupLocalPhaseUnspecified, phaseType(phase),
			"phase %q has no telemetry enum: add a case to phaseType and a value to "+
				"SetupLocalPhase (setup_local.go + universe proto)", phase)
	}
}
