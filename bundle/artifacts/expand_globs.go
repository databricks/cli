package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/patchwheel"
)

func createGlobError(v dyn.Value, p dyn.Path, message string) diag.Diagnostic {
	// The pattern contained in v is an absolute path.
	// Make it relative to the value's location to make it more readable.
	source := v.MustString()
	if l := v.Location(); l.File != "" {
		rel, err := filepath.Rel(filepath.Dir(l.File), source)
		if err == nil {
			source = rel
		}
	}

	return diag.Diagnostic{
		Severity:  diag.Error,
		Summary:   fmt.Sprintf("%s: %s", source, message),
		Locations: []dyn.Location{v.Location()},
		Paths:     []dyn.Path{p},
	}
}

type expandGlobs struct {
	name string
}

func (e expandGlobs) Name() string {
	return "expandGlobs"
}

func (e expandGlobs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Base path for this mutator.
	// This path is set with the list of expanded globs when done.
	base := dyn.NewPath(
		dyn.Key("artifacts"),
		dyn.Key(e.name),
		dyn.Key("files"),
	)

	// Pattern to match the source key in the files sequence.
	pattern := dyn.NewPatternFromPath(base).Append(
		dyn.AnyIndex(),
		dyn.Key("source"),
	)

	var diags diag.Diagnostics
	err := b.Config.Mutate(func(rootv dyn.Value) (dyn.Value, error) {
		var output []dyn.Value
		_, err := dyn.MapByPattern(rootv, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if v.Kind() != dyn.KindString {
				return v, nil
			}

			source := v.MustString()

			// Expand any glob reference in files source path
			matches, err := filepath.Glob(source)
			if err != nil {
				diags = diags.Append(createGlobError(v, p, err.Error()))

				// Drop this value from the list; this does not matter since we've raised an error anyway
				return v, nil
			}

			// Note, we're applying this for all artifact types, not just "whl".
			// Rationale:
			//  1. type is optional
			//  2. if you have wheels in other artifact type, maybe you still want the filter logic? impossible to say.
			matches = patchwheel.FilterLatestWheels(ctx, matches)

			if len(matches) == 1 && matches[0] == source {
				// No glob expansion was performed.
				// Keep node unchanged. We need to ensure that "patched" field remains and not wiped out by code below.
				parent, err := dyn.GetByPath(rootv, p[0:len(p)-1])
				if err != nil {
					log.Debugf(ctx, "Failed to get parent of %s", p.String())
				} else {
					output = append(output, parent)
				}
				return v, nil
			}

			if len(matches) == 0 {
				diags = diags.Append(createGlobError(v, p, "no matching files"))

				// Drop this value from the list; this does not matter since we've raised an error anyway
				return v, nil
			}

			for _, match := range matches {
				output = append(output, dyn.V(
					map[string]dyn.Value{
						"source": dyn.NewValue(match, v.Locations()),
					},
				))
			}

			return v, nil
		})

		if err != nil || diags.HasError() {
			return rootv, err
		}

		// Set the expanded globs back into the configuration.
		return dyn.SetByPath(rootv, base, dyn.V(output))
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}
