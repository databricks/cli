package libraries

import (
	"context"
	"maps"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/patchwheel"
)

type switchToPatchedWheels struct{}

func (c switchToPatchedWheels) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	replacements := getReplacements(ctx, b.Config.Artifacts, b.SyncRoot.Native())

	if len(replacements) == 0 {
		return nil
	}

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		var err error
		for _, pw := range patchedWheelPatterns() {
			v, err = dyn.MapByPattern(v, pw.pattern, func(p dyn.Path, wheel dyn.Value) (dyn.Value, error) {
				return switchToPatchedWheel(ctx, replacements, pw.withExtras, p, wheel), nil
			})
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		return v, nil
	})
	return diag.FromErr(err)
}

type patchedWheelPattern struct {
	pattern dyn.Pattern

	// withExtras says that a value may carry a pip extras suffix.
	withExtras bool
}

// patchedWheelPatterns returns the patterns of the wheels that are switched to
// patched wheels: the "whl" entries of the libraries in a list and every
// dependency in a list, except in locations that skip patched wheels. They are
// in the order of [locationsByKind], which is the order of the debug logs.
func patchedWheelPatterns() []patchedWheelPattern {
	var out []patchedWheelPattern
	for _, l := range locationsByKind() {
		if l.SkipPatchedWheels {
			continue
		}
		switch l.Kind {
		case resources.LibraryList:
			out = append(out, patchedWheelPattern{pattern: l.rootPattern.Append(dyn.AnyIndex(), dyn.Key("whl"))})
		case resources.DependencyList:
			out = append(out, patchedWheelPattern{pattern: l.rootPattern.Append(dyn.AnyIndex()), withExtras: true})
		case resources.LibraryFile:
			// Not switched; see [resources.LibraryFile].
		}
	}
	return out
}

func switchToPatchedWheel(ctx context.Context, replacements map[string]string, withExtras bool, p dyn.Path, v dyn.Value) dyn.Value {
	value, ok := v.AsString()
	if !ok {
		return v
	}

	base, extras := value, ""
	if withExtras {
		// A dependency may carry a pip extras suffix (e.g. "foo.whl[train]");
		// the patched-wheel replacements are keyed by the bare path, so strip
		// it for the lookup and re-append it to the patched path.
		base, extras = patchwheel.SplitWheelExtras(value)
	}

	repl := replacements[base]
	if repl == "" {
		log.Debugf(ctx, "Not updating %s from %s. Available replacements: %v", p, value, slices.Sorted(maps.Keys(replacements)))
		return v
	}

	log.Debugf(ctx, "Updating %s from %s to %s", p, value, repl+extras)
	return dyn.NewValue(repl+extras, v.Locations())
}

func getReplacements(ctx context.Context, artifacts config.Artifacts, root string) map[string]string {
	result := make(map[string]string)
	for _, artifact := range artifacts {
		for _, file := range artifact.Files {
			if file.Patched != "" {
				source, err := filepath.Rel(root, file.Source)
				if err != nil {
					log.Debugf(ctx, "Failed to get relative path (%s, %s): %s", root, file.Source, err)
					continue
				}
				patched, err := filepath.Rel(root, file.Patched)
				if err != nil {
					log.Debugf(ctx, "Failed to get relative path (%s, %s): %s", root, file.Patched, err)
					continue
				}
				result[source] = patched
				// There already was a check for duplicate by same_name_libraries.go
			}
		}
	}
	return result
}

func (c switchToPatchedWheels) Name() string {
	return "SwitchToPatchedWheels"
}

func SwitchToPatchedWheels() bundle.Mutator {
	return switchToPatchedWheels{}
}
