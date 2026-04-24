package phases

import "github.com/databricks/cli/ucm/deployplan"

// PlanOutcome is the engine-neutral result of the Plan phase. Both engines
// (terraform and direct) produce the same shape so cmd/ucm/plan.go renders
// identically regardless of engine.
type PlanOutcome struct {
	// Plan is the DAB-parity structured plan. Always non-nil on a successful
	// Plan call; when there are no changes, Plan.Plan is an empty map.
	Plan *deployplan.Plan

	// HasChanges is true when Plan contains at least one non-Skip action.
	HasChanges bool

	// Summary is the one-liner ("plan has changes" / "no changes") emitted
	// by the engine's output pipeline. Both engines produce byte-identical
	// strings so downstream consumers see stable output across engines.
	Summary string
}
