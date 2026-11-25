package direct

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dagrun"
)

// DetectDependencyCycle reports a cycle in the dependencies between the resources in configRoot
// (references and lifecycle.depends_on), the one CalculatePlan rejects. It builds the plan's graph
// with makePlan against empty state, so it reads nothing from state or the workspace, and finds
// only cycles the plan rejects too: with real state the graph only gains nodes. It returns nil
// when there is no cycle or the graph cannot be built; the plan reports those errors itself.
func DetectDependencyCycle(ctx context.Context, configRoot *config.Root) *dagrun.CycleError {
	adapters, err := dresources.InitAll(nil)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	// A bundle of its own, so the state cache makePlan fills is not the one deploy uses.
	b := DeploymentBundle{Adapters: adapters}
	plan, err := b.makePlan(ctx, configRoot, &dstate.Database{})
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	g, err := makeGraph(plan)
	if err != nil {
		return nil //nolint:nilerr // the plan reports this error
	}

	cycleErr, _ := errors.AsType[*dagrun.CycleError](g.DetectCycle())
	return cycleErr
}
