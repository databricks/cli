package validate

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/trampoline"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type referenceFields struct {
	bundle.RO
	afterBuild bool
}

// ReferenceFields rejects references to resource fields that do not exist, such as
// ${resources.volumes.bar.non_existent}, the same references the plan rejects. "bundle validate"
// runs it before the build; plan and deploy run [ReferenceFieldsAfterBuild] instead.
func ReferenceFields() bundle.ReadOnlyMutator {
	return &referenceFields{}
}

// ReferenceFieldsAfterBuild is [ReferenceFields] for plan and deploy. It runs after the build, which
// has already applied the python wheel wrapper, so it skips no wheel tasks, and it reads the open
// state, so it also checks empty resources that have state.
func ReferenceFieldsAfterBuild() bundle.ReadOnlyMutator {
	return &referenceFields{afterBuild: true}
}

func (*referenceFields) Name() string {
	return "validate:reference_fields"
}

func (m *referenceFields) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// The python wheel wrapper moves the wheel tasks it wraps and their wheel libraries, and the
	// references in them, out of the job during the build, so the plan does not resolve those
	// references.
	skipWheelTasks := !m.afterBuild && b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.PythonWheelWrapper)

	// Plan and deploy have the state open: the plan includes the resources in it even when they are
	// empty, and resolves their references.
	var db *dstate.Database
	if m.afterBuild {
		db = &b.DeploymentBundle.StateDB.Data
	}

	found, err := direct.FindUnknownReferenceFields(&b.Config, db)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for _, f := range found {
		if skipWheelTasks && isInWheelTask(f.Path) && mayBeWrapped(b, f.Path) {
			continue
		}
		d := diag.Errorf("cannot resolve %q: %s", "${"+f.Reference+"}", f.Err)
		d[0].Locations = f.Value.Locations()
		d[0].Paths = dyn.ToStructPaths(f.Path)
		diags = append(diags, d...)
	}
	return diags
}

// isInWheelTask reports whether path is in a job task field that the python wheel wrapper moves out
// of the tasks it wraps.
func isInWheelTask(path dyn.Path) bool {
	if len(path) < 6 || path[0].Key() != "resources" || path[1].Key() != "jobs" || path[3].Key() != "tasks" {
		return false
	}
	return trampoline.IsMovedTaskField(path[5:])
}

// mayBeWrapped reports whether the python wheel wrapper may wrap the job task at path. The wrapper
// decides after the build, which may upload a local library to the workspace, so a Python wheel
// task with a local library may be wrapped too.
func mayBeWrapped(b *bundle.Bundle, path dyn.Path) bool {
	task := b.Config.Resources.Jobs[path[2].Key()].Tasks[path[4].Index()]
	return trampoline.WrapsTask(task) || (task.PythonWheelTask != nil && libraries.IsTaskWithLocalLibraries(task))
}
