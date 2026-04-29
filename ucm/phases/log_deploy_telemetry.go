package phases

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/ucm"
)

// ResourceIdLimit caps how many per-resource IDs we attach to a deploy
// telemetry event. Mirrors bundle.phases.ResourceIdLimit; declared here
// rather than imported so ucm/** carries no bundle/** dependency.
//
// No ucm resource kind currently emits IDs into the telemetry event (see
// LogDeployTelemetry's package comment), so this constant is exported for
// parity and future use rather than read in this file today.
const ResourceIdLimit = 1000

// maxErrorMessageLength caps the scrubbed error message attached to deploy
// telemetry. Mirrors the same constant in bundle/phases/telemetry.go.
const maxErrorMessageLength = 500

// getExecutionTimes mirrors bundle/phases.getExecutionTimes: sort the
// per-mutator execution times in descending order and keep the top 250
// entries to keep the telemetry event size bounded regardless of how
// many mutators ran.
func getExecutionTimes(u *ucm.Ucm) []protos.IntMapEntry {
	executionTimes := u.Metrics.ExecutionTimes

	slices.SortFunc(executionTimes, func(a, b protos.IntMapEntry) int {
		return cmp.Compare(b.Value, a.Value)
	})

	if len(executionTimes) > 250 {
		executionTimes = executionTimes[:250]
	}

	return executionTimes
}

// LogDeployTelemetry logs a telemetry event for a ucm deploy command.
//
// Forked from bundle/phases.LogDeployTelemetry. The wire format reuses
// protos.BundleDeployEvent — a dedicated UcmDeployEvent would require a
// libs/telemetry/protos edit, which is owned by upstream and out of scope
// here. ucm fills the proto fields that have ucm analogues and leaves
// the rest at their zero values.
//
// UCM-specific deviations from the bundle implementation:
//   - bundle.Bundle.Uuid: ucm has no top-level uuid; the event records
//     the all-zero UUID as bundle does in the unset case.
//   - bundle.Metrics.DeploymentId / ConfigurationFileCount / TargetCount /
//     BoolValues / LocalCacheMeasurementsMs / Python* are zero — ucm
//     does not collect those metrics today. They will populate naturally
//     as the matching plumbing lands.
//   - bundle.Workspace.ArtifactPath, bundle.Bundle.Mode, the python /
//     experimental config blocks: no ucm equivalent, reported as
//     "unspecified".
//   - The proto's resource_*_ids slots are jobs / pipelines / clusters /
//     dashboards — none of which are ucm resource kinds. Catalog,
//     schema, volume, etc. names are PII (matching bundle's stance on
//     schemas and volumes), so ucm emits no per-resource IDs into this
//     event today; resource counts still reflect the declared
//     configuration.
func LogDeployTelemetry(ctx context.Context, u *ucm.Ucm, errMsg string) {
	errMsg = scrubForTelemetry(errMsg)

	if len(errMsg) > maxErrorMessageLength {
		errMsg = errMsg[:maxErrorMessageLength]
	}

	resourcesCount := int64(0)
	_, err := dyn.MapByPattern(u.Config.Value(), dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		resourcesCount++
		return v, nil
	})
	if err != nil {
		log.Debugf(ctx, "failed to count resources: %s", err)
	}

	resources := u.Config.Resources

	variableCount := len(u.Config.Variables)
	complexVariableCount := int64(0)
	lookupVariableCount := int64(0)
	for _, v := range u.Config.Variables {
		// Resolved value drives complexity classification — the optional
		// "type: complex" annotation can be omitted in ucm.yml.
		if v.IsComplexValued() {
			complexVariableCount++
		}
		if v.Lookup != nil {
			lookupVariableCount++
		}
	}

	telemetry.Log(ctx, protos.DatabricksCliLog{
		BundleDeployEvent: &protos.BundleDeployEvent{
			BundleUuid:   "00000000-0000-0000-0000-000000000000",
			ErrorMessage: errMsg,

			ResourceCount:       resourcesCount,
			ResourceSchemaCount: int64(len(resources.Schemas)),
			ResourceVolumeCount: int64(len(resources.Volumes)),

			Experimental: &protos.BundleDeployExperimental{
				BundleMode:                   protos.BundleModeUnspecified,
				WorkspaceArtifactPathType:    protos.BundleDeployArtifactPathTypeUnspecified,
				VariableCount:                int64(variableCount),
				ComplexVariableCount:         complexVariableCount,
				LookupVariableCount:          lookupVariableCount,
				BundleMutatorExecutionTimeMs: getExecutionTimes(u),
			},
		},
	})
}
