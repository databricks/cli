package artifacts

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type expandGlobs struct {
	name string
}

func (m *expandGlobs) Name() string {
	return fmt.Sprintf("artifacts.ExpandGlobs(%s)", m.name)
}

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

func (m *expandGlobs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Base path for this mutator.
	// This path is set with the list of expanded globs when done.
	base := dyn.NewPath(
		dyn.Key("artifacts"),
		dyn.Key(m.name),
		dyn.Key("files"),
	)

	// Pattern to match the source key in the files sequence.
	pattern := dyn.NewPatternFromPath(base).Append(
		dyn.AnyIndex(),
		dyn.Key("source"),
	)

	var diags diag.Diagnostics
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var output []dyn.Value
		_, err := dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if v.Kind() != dyn.KindString {
				return v, nil
			}

			source := v.MustString()

			// Expand any glob reference in files source path
			matches, err := filepath.Glob(source)
			if err != nil {
				diags = diags.Append(createGlobError(v, p, err.Error()))

				// Continue processing and leave this value unchanged.
				return v, nil
			}

			if len(matches) == 0 {
				diags = diags.Append(createGlobError(v, p, "no matching files"))

				// Continue processing and leave this value unchanged.
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
			return v, err
		}

		// Set the expanded globs back into the configuration.
		return dyn.SetByPath(v, base, dyn.V(output))
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return diags
}
