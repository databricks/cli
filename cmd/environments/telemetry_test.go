package environments

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("no-constraints run records skipConstraints", func(t *testing.T) {
		res := &libslocalenv.Result{
			OK:              true,
			Mode:            libslocalenv.ModeDefault.String(),
			SkipConstraints: true,
			Compute:         &libslocalenv.ComputeInfo{Source: "cluster", EnvKey: "dbr/16.4"},
		}
		got := buildSetupLocalEvent(res)
		assert.True(t, got.SkipConstraints)
	})

	t.Run("run without --no-constraints leaves skipConstraints false", func(t *testing.T) {
		res := &libslocalenv.Result{
			OK:      true,
			Mode:    libslocalenv.ModeDefault.String(),
			Compute: &libslocalenv.ComputeInfo{Source: "cluster", EnvKey: "dbr/16.4"},
		}
		got := buildSetupLocalEvent(res)
		assert.False(t, got.SkipConstraints)
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

// TestErrorCodeCoversLocalenv asserts every localenv.ErrorCode maps to a
// non-unspecified telemetry enum. Completeness of the switch itself is enforced
// by the exhaustive linter (errorCodeType has no default); this test additionally
// catches a case that maps to UNSPECIFIED by mistake. Keep this list in sync with
// the const block in libs/localenv/result.go.
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

// TestPhaseCoversLocalenv asserts every localenv.PhaseName maps to a
// non-unspecified telemetry enum. As with TestErrorCodeCoversLocalenv, switch
// completeness is enforced by the exhaustive linter; this guards against a case
// mapping to UNSPECIFIED by mistake.
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

func TestLogSetupLocalEventRoundTrip(t *testing.T) {
	t.Run("real run uploads one setup_local_event", func(t *testing.T) {
		server := testserver.New(t)
		t.Cleanup(server.Close)

		var bodies []telemetry.RequestBody
		server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
			var body telemetry.RequestBody
			require.NoError(t, json.Unmarshal(req.Body, &body))
			bodies = append(bodies, body)
			return telemetry.ResponseBody{NumProtoSuccess: int64(len(body.ProtoLogs))}
		})

		ctx := telemetry.WithNewLogger(t.Context())
		logSetupLocalEvent(ctx, &libslocalenv.Result{
			OK:      true,
			Mode:    libslocalenv.ModeDefault.String(),
			Compute: &libslocalenv.ComputeInfo{Source: "serverless", EnvKey: "serverless/serverless-v5"},
		})

		ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})
		require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))

		require.Len(t, bodies, 1)
		require.Len(t, bodies[0].ProtoLogs, 1)
		assert.Contains(t, bodies[0].ProtoLogs[0], `"setup_local_event"`)
		assert.Contains(t, bodies[0].ProtoLogs[0], `"env_key":"serverless/serverless-v5"`)
	})

	t.Run("dry-run buffers nothing to upload", func(t *testing.T) {
		called := false
		server := testserver.New(t)
		t.Cleanup(server.Close)
		server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
			called = true
			return telemetry.ResponseBody{}
		})

		ctx := telemetry.WithNewLogger(t.Context())
		logSetupLocalEvent(ctx, &libslocalenv.Result{DryRun: true, OK: true})

		ctx = cmdctx.SetConfigUsed(ctx, &config.Config{Host: server.URL, Token: "token"})
		// Upload no-ops when the buffer is empty, so the endpoint is never hit.
		require.NoError(t, telemetry.Upload(ctx, protos.ExecutionContext{}))
		assert.False(t, called, "dry-run must not upload any telemetry")
	})
}
