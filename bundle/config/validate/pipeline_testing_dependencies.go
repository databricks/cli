package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// pipelineTestingDependencyFlags are the pip flags that a test dependency can put before its path.
var pipelineTestingDependencyFlags = []string{"-r", "--requirement", "-e", "--editable"}

type pipelineTestingDependencies struct{ bundle.RO }

// PipelineTestingDependencies rejects entries of testing.environment.dependencies that do not
// point at a file, such as package names and URLs. The API rejects them too, but only on deploy.
func PipelineTestingDependencies() bundle.ReadOnlyMutator {
	return &pipelineTestingDependencies{}
}

func (v *pipelineTestingDependencies) Name() string {
	return "validate:pipeline_testing_dependencies"
}

func (v *pipelineTestingDependencies) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(),
		dyn.Key("testing"), dyn.Key("environment"), dyn.Key("dependencies"), dyn.AnyIndex(),
	)
	_, err := dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		dependency, ok := v.AsString()
		if !ok || isPipelineTestingFileDependency(dependency) {
			return v, nil
		}
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("testing.environment.dependencies must point at files, not %q", dependency),
			Detail:    "Declare test packages in a project file, such as the test extra of a pyproject.toml, and reference that file, for example --editable ${workspace.file_path}[test].",
			Locations: v.Locations(),
			Paths:     dyn.ToStructPaths(p),
		})
		return v, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}
	return diags
}

// isPipelineTestingFileDependency reports whether dependency is a workspace or volume path, or a
// local file that deploy uploads, with an optional pip flag before it.
func isPipelineTestingFileDependency(dependency string) bool {
	// A reference resolved at deploy time, such as a volume path, is checked by the API.
	if dynvar.ContainsVariableReference(dependency) {
		return true
	}
	for _, flag := range pipelineTestingDependencyFlags {
		if path, ok := strings.CutPrefix(dependency, flag+" "); ok {
			dependency = strings.TrimSpace(path)
			break
		}
	}
	return strings.HasPrefix(dependency, "/") || libraries.IsLibraryLocal(dependency)
}
