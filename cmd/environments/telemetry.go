package environments

import (
	"context"

	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// logSetupLocalEvent buffers a setup-local telemetry event for a real run;
// cmd/root uploads it at exit. It is a no-op for dry-run (buildSetupLocalEvent
// returns nil), so callers do not need to guard on res.DryRun.
func logSetupLocalEvent(ctx context.Context, res *libslocalenv.Result) {
	event := buildSetupLocalEvent(res)
	if event == nil {
		return
	}
	telemetry.Log(ctx, protos.DatabricksCliLog{SetupLocalEvent: event})
}

// buildSetupLocalEvent maps a pipeline Result to the telemetry event, or returns
// nil when there is nothing to report (nil result, or a --dry-run run, which is
// not recorded). The Result is already PII-safe: it backs the --output json
// contract, so only abstracted fields (source, mode, env key, error class) are
// read here — never cluster names/ids or paths.
func buildSetupLocalEvent(res *libslocalenv.Result) *protos.SetupLocalEvent {
	if res == nil || res.DryRun {
		return nil
	}
	event := &protos.SetupLocalEvent{
		Success:         res.OK,
		Mode:            modeType(res.Mode),
		SkipConstraints: res.SkipConstraints,
		Greenfield:      res.Greenfield,
		ComputeSource:   protos.SetupLocalComputeSourceUnspecified,
	}
	if res.Compute != nil {
		event.ComputeSource = computeSourceType(res.Compute.Source)
		event.EnvKey = res.Compute.EnvKey
	}
	if res.Error != nil {
		event.ErrorCode = errorCodeType(res.Error.Code)
		event.FailurePhase = phaseType(res.Error.FailurePhase)
		event.DiskMutated = res.Error.DiskMutated
	}
	return event
}

// computeSourceType maps a resolved ComputeInfo.Source to its telemetry enum.
func computeSourceType(source string) protos.SetupLocalComputeSource {
	switch source {
	case "cluster":
		return protos.SetupLocalComputeSourceCluster
	case "serverless":
		return protos.SetupLocalComputeSourceServerless
	case "job":
		return protos.SetupLocalComputeSourceJob
	case "bundle":
		return protos.SetupLocalComputeSourceBundle
	default:
		return protos.SetupLocalComputeSourceUnspecified
	}
}

// modeType maps the Result.Mode string (Mode.String()) to its telemetry enum.
func modeType(mode string) protos.SetupLocalMode {
	switch mode {
	case libslocalenv.ModeDefault.String():
		return protos.SetupLocalModeDefault
	case libslocalenv.ModeConstraintsOnly.String():
		return protos.SetupLocalModeConstraintsOnly
	default:
		return protos.SetupLocalModeUnspecified
	}
}

// errorCodeType maps a pipeline ErrorCode to its telemetry enum. The switch has
// no default, so the exhaustive linter fails the build if a new
// localenv.ErrorCode is added without a case here, guarding against silently
// logging a new code as UNSPECIFIED. The trailing return handles a value outside
// the declared set (a newer localenv than this mapping).
func errorCodeType(code libslocalenv.ErrorCode) protos.SetupLocalErrorCode {
	switch code {
	case libslocalenv.ErrUsage:
		return protos.SetupLocalErrorCodeUsage
	case libslocalenv.ErrManagerUnsupported:
		return protos.SetupLocalErrorCodeManagerUnsupported
	case libslocalenv.ErrNotWritable:
		return protos.SetupLocalErrorCodeNotWritable
	case libslocalenv.ErrUvMissing:
		return protos.SetupLocalErrorCodeUvMissing
	case libslocalenv.ErrNoTarget:
		return protos.SetupLocalErrorCodeNoTarget
	case libslocalenv.ErrResolve:
		return protos.SetupLocalErrorCodeResolve
	case libslocalenv.ErrEnvUnsupported:
		return protos.SetupLocalErrorCodeEnvUnsupported
	case libslocalenv.ErrFetch:
		return protos.SetupLocalErrorCodeFetch
	case libslocalenv.ErrWrite:
		return protos.SetupLocalErrorCodeWrite
	case libslocalenv.ErrMerge:
		return protos.SetupLocalErrorCodeMerge
	case libslocalenv.ErrPythonInstall:
		return protos.SetupLocalErrorCodePythonInstall
	case libslocalenv.ErrProvision:
		return protos.SetupLocalErrorCodeProvision
	case libslocalenv.ErrValidate:
		return protos.SetupLocalErrorCodeValidate
	case libslocalenv.ErrCanceled:
		return protos.SetupLocalErrorCodeCanceled
	}
	return protos.SetupLocalErrorCodeUnspecified
}

// phaseType maps a pipeline PhaseName to its telemetry enum. As with
// errorCodeType, the switch has no default so the exhaustive linter fails the
// build if a new localenv.PhaseName is added without a case here.
func phaseType(phase libslocalenv.PhaseName) protos.SetupLocalPhase {
	switch phase {
	case libslocalenv.PhasePreflight:
		return protos.SetupLocalPhasePreflight
	case libslocalenv.PhaseResolve:
		return protos.SetupLocalPhaseResolve
	case libslocalenv.PhaseFetch:
		return protos.SetupLocalPhaseFetch
	case libslocalenv.PhaseMerge:
		return protos.SetupLocalPhaseMerge
	case libslocalenv.PhaseProvision:
		return protos.SetupLocalPhaseProvision
	case libslocalenv.PhaseValidate:
		return protos.SetupLocalPhaseValidate
	}
	return protos.SetupLocalPhaseUnspecified
}
