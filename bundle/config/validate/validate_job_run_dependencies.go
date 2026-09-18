package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

func ValidateJobRunDependencies() bundle.ReadOnlyMutator {
	return &validateJobRunDependencies{}
}

type validateJobRunDependencies struct{ bundle.RO }

func (v *validateJobRunDependencies) Name() string {
	return "validate:job_run_dependencies"
}

func (v *validateJobRunDependencies) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, name := range slices.Sorted(maps.Keys(b.Config.Resources.JobRuns)) {
		jobRun := b.Config.Resources.JobRuns[name]
		if jobRun == nil {
			continue
		}

		for i, dependency := range jobRun.DependsOn {
			path := dyn.NewPath(
				dyn.Key("resources"),
				dyn.Key("job_runs"),
				dyn.Key(name),
				dyn.Key("depends_on"),
				dyn.Index(i),
			)
			refPath, ok := dynvar.PureReferenceToPath(dependency)
			if !ok || !isJobRunIDReference(refPath) {
				diags = append(diags, invalidJobRunDependency(b, path))
				continue
			}

			target := refPath[2].Key()
			if _, ok := b.Config.Resources.JobRuns[target]; !ok {
				diags = append(diags, diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("depends_on references undefined job run %q", target),
					Paths:     []dyn.Path{path},
					Locations: b.Config.GetLocations(path.String()),
				})
			}
		}
	}

	return diags
}

func isJobRunIDReference(path dyn.Path) bool {
	return len(path) == 4 &&
		path[0].Key() == "resources" &&
		path[1].Key() == "job_runs" &&
		path[2].Key() != "" &&
		path[3].Key() == "id"
}

func invalidJobRunDependency(b *bundle.Bundle, path dyn.Path) diag.Diagnostic {
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   "depends_on entries must be job run ID references, for example ${resources.job_runs.prepare.id}",
		Paths:     []dyn.Path{path},
		Locations: b.Config.GetLocations(path.String()),
	}
}
