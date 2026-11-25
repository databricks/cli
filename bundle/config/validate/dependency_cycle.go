package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

type dependencyCycle struct{ bundle.RO }

// DependencyCycle rejects resources whose references or lifecycle.depends_on form a cycle. The
// plan rejects them too, but only after deploy has uploaded the bundle files.
func DependencyCycle() bundle.ReadOnlyMutator {
	return &dependencyCycle{}
}

func (*dependencyCycle) Name() string {
	return "validate:dependency_cycle"
}

func (*dependencyCycle) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	cycle := direct.DetectDependencyCycle(&b.Config)
	if cycle == nil {
		return nil
	}

	d := diag.Diagnostic{
		Severity: diag.Error,
		Summary:  cycle.Error(),
	}
	// Point at the resource the message names first: the one holding the reference that starts
	// the cycle (in a cycle of one, Nodes[0] refers to itself).
	if len(cycle.Nodes) > 0 {
		node := cycle.Nodes[min(1, len(cycle.Nodes)-1)]
		if path, err := structpath.ParsePath(node); err == nil {
			d.Locations = b.Config.GetLocationsOf(path)
			d.Paths = []*structpath.PathNode{path}
		}
	}
	return diag.Diagnostics{d}
}
