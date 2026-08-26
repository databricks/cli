package mutator

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
)

type resolveJobRunValueTriggers struct{}

// ResolveJobRunValueTriggers resolves the bundle/workspace/variable references in
// each on_value_change expression and records expression → value on the job_run.
// ${resources.*} is left for the planner, which resolves it after apply.
func ResolveJobRunValueTriggers() bundle.Mutator {
	return &resolveJobRunValueTriggers{}
}

func (*resolveJobRunValueTriggers) Name() string {
	return "ResolveJobRunValueTriggers"
}

func (*resolveJobRunValueTriggers) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var normalized dyn.Value
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		// IncludeMissingFields so a reference to an unset field resolves to the
		// empty value rather than failing, as in ResolveVariableReferences.
		normalized, _ = convert.Normalize(b.Config, root, convert.IncludeMissingFields)
		return root, nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	varPath := dyn.NewPath(dyn.Key("var"))
	prefixes := make([]dyn.Path, len(defaultPrefixes))
	for i, prefix := range defaultPrefixes {
		prefixes[i] = dyn.MustPathFromString(prefix)
	}

	var diags diag.Diagnostics
	for name, jr := range b.Config.Resources.JobRuns {
		if jr == nil || jr.Lifecycle == nil {
			continue
		}

		// Expressions are unique: ValidateJobRunTriggers rejects duplicates.
		out := make(map[string]string)
		for i, t := range jr.Lifecycle.Triggers {
			if t.OnValueChange == nil {
				continue
			}

			expr := strings.TrimSpace(*t.OnValueChange)
			value, err := resolveJobRunValueTrigger(b, normalized, prefixes, varPath, expr)
			if err != nil {
				path := fmt.Sprintf("resources.job_runs.%s.lifecycle.triggers[%d].on_value_change", name, i)
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("lifecycle.triggers.on_value_change: %s", err),
					Locations: b.Config.GetLocations(path),
				})
				continue
			}
			out[expr] = value
		}

		if len(out) > 0 {
			jr.ResolvedValueTriggers = out
		}
	}

	return diags
}

func resolveJobRunValueTrigger(b *bundle.Bundle, normalized dyn.Value, prefixes []dyn.Path, varPath dyn.Path, expr string) (string, error) {
	resolved, err := dynvar.Resolve(dyn.V(expr), func(path dyn.Path) (dyn.Value, error) {
		// Rewrite the shorthand path ${var.foo} into ${variables.foo.value}.
		if path.HasPrefix(varPath) {
			newPath := dyn.NewPath(dyn.Key("variables"), path[1], dyn.Key("value"))
			if len(path) > 2 {
				newPath = newPath.Append(path[2:]...)
			}
			path = newPath
		}

		if slices.ContainsFunc(prefixes, path.HasPrefix) {
			return lookup(normalized, path, b)
		}

		return dyn.InvalidValue, dynvar.ErrSkipResolution
	})
	if err != nil {
		return "", err
	}

	// A whole-string reference resolves to the referenced type, so a non-string
	// value (e.g. a numeric variable) still needs a textual fingerprint.
	if s, ok := resolved.AsString(); ok {
		return s, nil
	}

	return structaccess.ConvertToString(resolved.AsAny())
}
