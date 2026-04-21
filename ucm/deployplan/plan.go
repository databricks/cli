package deployplan

import (
	"cmp"
	"slices"

	"github.com/databricks/cli/internal/build"
)

// Plan is the ucm deployment plan, deliberately kept shape-compatible with
// bundle/deployplan.Plan for the fields Phase 1 exercises. Direct-engine
// fields (DependsOn, NewState, RemoteState, Changes, lineage, serial) are
// left off until Phase 2 needs them.
type Plan struct {
	PlanVersion int                   `json:"plan_version,omitempty"`
	CLIVersion  string                `json:"cli_version,omitempty"`
	Plan        map[string]*PlanEntry `json:"plan,omitzero"`
}

type PlanEntry struct {
	Action ActionType `json:"action,omitempty"`
}

// NewPlanTerraform creates a Plan for terraform-engine output. Mirrors
// bundle/deployplan.NewPlanTerraform: no plan_version (terraform-backed plans
// are not loaded back from disk as direct-engine plans are).
func NewPlanTerraform() *Plan {
	return &Plan{
		CLIVersion: build.GetInfo().Version,
		Plan:       make(map[string]*PlanEntry),
	}
}

// GetActions returns the plan's entries sorted by resource key.
func (p *Plan) GetActions() []Action {
	actions := make([]Action, 0, len(p.Plan))
	for key, entry := range p.Plan {
		actions = append(actions, Action{
			ResourceKey: key,
			ActionType:  entry.Action,
		})
	}
	slices.SortFunc(actions, func(x, y Action) int {
		return cmp.Compare(x.ResourceKey, y.ResourceKey)
	})
	return actions
}
