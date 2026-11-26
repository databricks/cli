package direct

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dagrun"
	"github.com/databricks/cli/libs/dyn"
)

// DetectDependencyCycle reports a cycle in the dependencies between the resources in configRoot
// (references and lifecycle.depends_on), the one CalculatePlan rejects. It builds the plan's graph
// with makePlan against db (empty state when nil) without reading the workspace, and finds only
// cycles the plan rejects too: the plan against the full state only adds nodes and edges. It
// returns nil when there is no cycle or the graph cannot be built; the plan reports those errors.
// It does not modify configRoot or db.
func DetectDependencyCycle(ctx context.Context, configRoot *config.Root, db *dstate.Database) *dagrun.CycleError {
	// makePlan reads each resource back by its key's string form and panics on a key that does
	// not survive it; leave such configs to the plan.
	if hasKeyNotReadableByString(configRoot.Value()) {
		return nil
	}

	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	// A bundle of its own, so the state cache makePlan fills is not the one deploy uses.
	b := DeploymentBundle{Adapters: adapters}
	if db == nil {
		db = &dstate.Database{}
	}
	plan, err := b.makePlan(ctx, configRoot, db)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	// For an entry that is not deleted, an edge to a node outside the plan fails makeGraph, but it
	// can only be a reference the plan resolves or rejects later (${artifacts...} before the build,
	// a missing resource), so drop it to still find the cycles among the other nodes. Delete
	// entries share their DependsOn with db, so build new slices instead of filtering in place.
	for _, entry := range plan.Plan {
		var deps []deployplan.DependsOnEntry
		for _, dep := range entry.DependsOn {
			if plan.Plan[dep.Node] != nil {
				deps = append(deps, dep)
			}
		}
		entry.DependsOn = deps
	}

	g, err := makeGraph(plan)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	cycleErr, _ := errors.AsType[*dagrun.CycleError](g.DetectCycle())
	return cycleErr
}

// hasKeyNotReadableByString reports whether a resource key changes when its path goes through
// its string form, as a key with a dot does.
func hasKeyNotReadableByString(root dyn.Value) bool {
	found := false
	pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
	_, _ = dyn.MapByPattern(root, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		parsed, err := dyn.NewPathFromString(p.String())
		if err != nil || !parsed.Equal(p) {
			found = true
		}
		return v, nil
	})
	return found
}
