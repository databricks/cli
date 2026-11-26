package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type referenceFields struct {
	bundle.RO
	afterBuild bool
}

// ReferenceFields rejects references to resource fields that do not exist, such as
// ${resources.volumes.bar.non_existent}. The plan rejects them too, but only after deploy has
// uploaded the files.
func ReferenceFields() bundle.ReadOnlyMutator {
	return &referenceFields{}
}

// ReferenceFieldsAfterBuild is [ReferenceFields] for plan and deploy, run after the build, so it
// checks the same config as the plan.
func ReferenceFieldsAfterBuild() bundle.ReadOnlyMutator {
	return &referenceFields{afterBuild: true}
}

func (*referenceFields) Name() string {
	return "validate:reference_fields"
}

func (m *referenceFields) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	// The python wheel wrapper moves wheel tasks and their wheel libraries, and the references in
	// them, out of the job during the build, so the plan does not resolve those references.
	skipWheelTasks := !m.afterBuild && b.Config.Experimental != nil && config.IsExplicitlyEnabled(b.Config.Experimental.PythonWheelWrapper)

	found, err := direct.FindUnknownReferenceFields(&b.Config)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	for _, f := range found {
		if skipWheelTasks && isInWheelTask(f.Path) {
			continue
		}
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("cannot resolve \"${%s}\": %s", f.Reference, f.Err),
			Locations: f.Value.Locations(),
			Paths:     dyn.ToStructPaths(f.Path),
		})
	}
	return diags
}

// isInWheelTask reports whether path is in the python_wheel_task or the libraries of a job task,
// the fields the python wheel wrapper moves out of the job.
func isInWheelTask(path dyn.Path) bool {
	if len(path) < 6 || path[0].Key() != "resources" || path[1].Key() != "jobs" || path[3].Key() != "tasks" {
		return false
	}
	field := path[5].Key()
	return field == "python_wheel_task" || field == "libraries"
}
