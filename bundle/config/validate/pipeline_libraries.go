package validate

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// errorForIncompletePipelineLibraries rejects file, notebook, and glob entries
// without paths because the pipelines API rejects them.
func errorForIncompletePipelineLibraries(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for key, pipeline := range b.Config.Resources.Pipelines {
		for i, library := range pipeline.Libraries {
			base := dyn.NewPath(
				dyn.Key("resources"),
				dyn.Key("pipelines"),
				dyn.Key(key),
				dyn.Key("libraries"),
				dyn.Index(i),
			)
			if library.File != nil && strings.TrimSpace(library.File.Path) == "" {
				diags = diags.Append(libraryPathDiag(b, base, "file", "pipeline library file path is required"))
			}
			if library.Notebook != nil && strings.TrimSpace(library.Notebook.Path) == "" {
				diags = diags.Append(libraryPathDiag(b, base, "notebook", "pipeline library notebook path is required"))
			}
			if library.Glob != nil && strings.TrimSpace(library.Glob.Include) == "" {
				diags = diags.Append(libraryPathDiag(b, base, "glob", "pipeline library glob include is required"))
			}
		}
	}
	return diags
}

// libraryPathDiag reports a missing path on one pipeline library variant.
func libraryPathDiag(b *bundle.Bundle, base dyn.Path, field, summary string) diag.Diagnostic {
	fieldPath := base.Append(dyn.Key(field))
	locations := locationsAtPath(b, fieldPath)
	if len(locations) == 0 {
		locations = locationsAtPath(b, base)
	}
	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   summary,
		Locations: locations,
		Paths:     []dyn.Path{fieldPath},
	}
}
