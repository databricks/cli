package direct

import (
	"context"
	"errors"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dagrun"
)

// DetectDependencyCycle reports a cycle in the dependencies between the resources in configRoot
// (references and lifecycle.depends_on), the one CalculatePlan rejects. It builds the plan's graph
// with makePlan against db (empty state when nil) without reading the workspace, and finds only
// cycles the plan rejects too: the plan against the full state only adds nodes and edges. It
// returns nil when there is no cycle or the graph cannot be built; the plan reports those errors.
func DetectDependencyCycle(ctx context.Context, configRoot *config.Root, db *dstate.Database) (cycleErr *dagrun.CycleError) {
	// makePlan panics on some configs it cannot read (a resource key with a dot); the plan
	// itself fails on those, and a check that runs earlier must not crash on them first.
	defer func() {
		if recover() != nil {
			cycleErr = nil
		}
	}()

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

	// An edge to a node outside the plan fails makeGraph, but it can only be a reference the
	// plan resolves or rejects later (${artifacts...} before the build, a missing resource), so
	// drop it to still find the cycles among the other nodes.
	for _, entry := range plan.Plan {
		entry.DependsOn = slices.DeleteFunc(entry.DependsOn, func(dep deployplan.DependsOnEntry) bool {
			return plan.Plan[dep.Node] == nil
		})
	}

	g, err := makeGraph(plan)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	cycleErr, _ = errors.AsType[*dagrun.CycleError](g.DetectCycle())
	return cycleErr
}
