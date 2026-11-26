package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type dependencyCycle struct {
	bundle.RO
	afterBuild bool
}

// DependencyCycle rejects resources whose references or lifecycle.depends_on form a cycle, for
// "bundle validate". The plan rejects them too, but only after deploy has uploaded the files.
func DependencyCycle() bundle.ReadOnlyMutator {
	return &dependencyCycle{}
}

// DependencyCycleAfterBuild is [DependencyCycle] for plan and deploy, run after every step that
// changes the config before the plan, so it checks the same config as the plan.
func DependencyCycleAfterBuild() bundle.ReadOnlyMutator {
	return &dependencyCycle{afterBuild: true}
}

func (*dependencyCycle) Name() string {
	return "validate:dependency_cycle"
}

func (m *dependencyCycle) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// The python wheel wrapper moves wheel task parameters, and the references in them, out of
	// the job during the build, so before the build the config can have a cycle the plan does
	// not see.
	if !m.afterBuild && b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.PythonWheelWrapper) {
		return nil
	}

	// After the build, plan and deploy have the state open: planning against it also covers
	// resources kept only because they are in state.
	var db *dstate.Database
	if m.afterBuild && b.DeploymentBundle.StateDB.IsOpen() {
		db = &b.DeploymentBundle.StateDB.Data
	}
	cycle := direct.DetectDependencyCycle(ctx, &b.Config, db)
	if cycle == nil {
		return nil
	}

	diags := diag.Errorf("%s", cycle.Error())
	// Point at the resource the message names first: the one holding the reference that starts
	// the cycle (in a cycle of one, Nodes[0] refers to itself).
	node := cycle.Nodes[min(1, len(cycle.Nodes)-1)]
	if path, err := structpath.ParsePath(node); err == nil {
		diags[0].Locations = b.Config.GetLocationsOf(path)
		diags[0].Paths = []*structpath.PathNode{path}
	}
	return diags
}
