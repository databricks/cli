package phases

import "github.com/databricks/cli/ucm/deployplan"

// PlanOutcome is the engine-neutral result of the Plan phase: the structured
// plan itself plus a pair of legacy summary fields (HasChanges / Summary)
// preserved so existing callers and tests don't need to count plan entries
// manually. Both engines (terraform and direct) produce PlanOutcome so cmd
// /ucm/plan.go can render the same way regardless of engine.
type PlanOutcome struct {
	// Plan is the DAB-parity structured plan. Always non-nil on a successful
	// Plan call, even when there are no changes — in that case Plan.Plan is
	// an empty map.
	Plan *deployplan.Plan

	// HasChanges is true when Plan contains at least one non-Skip action.
	HasChanges bool

	// Summary is the legacy one-liner ("plan has changes" / "no changes")
	// emitted by the terraform engine's output pipeline. The direct engine
	// produces the same two strings so downstream consumers of Summary see
	// byte-identical output across engines.
	Summary string
}
