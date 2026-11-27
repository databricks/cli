package mutator

import (
	"context"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
)

type resolveResourceReferencesForDisplay struct{}

// ResolveResourceReferencesForDisplay resolves ${resources.*} references in every
// resource from the config, for read-only commands that display resources (summary,
// open), so names and URLs show values instead of references. Run it after
// statemgmt.Load, which sets the ids of deployed resources.
//
// Display is best effort, so a reference stays as it is when its value is not known
// or not safe to show:
//   - a field that is not set in the config, for example the id of a resource that
//     was never deployed (an implied empty value would turn an id into "");
//   - a sensitive field, such as a secret's value. Sensitive variables can only be
//     used in sensitive fields (validate.SensitiveVariableReferences), so other fields
//     hold no secrets;
//   - a resource that fails to resolve (a reference cycle) or whose resolved values
//     don't fit its type, such as a name in an int field.
func ResolveResourceReferencesForDisplay() bundle.Mutator {
	return &resolveResourceReferencesForDisplay{}
}

func (m *resolveResourceReferencesForDisplay) Name() string {
	return "ResolveResourceReferencesForDisplay"
}

var resourcesPath = dyn.NewPath(dyn.Key("resources"))

func (m *resolveResourceReferencesForDisplay) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		lookup := func(path dyn.Path) (dyn.Value, error) {
			// A pure reference to a whole resource (len 3) could include its sensitive fields.
			if len(path) < 4 || !path.HasPrefix(resourcesPath) || dresources.IsSensitiveResourceField(path) {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}
			v, err := dyn.GetByPath(root, path)
			if err != nil || !v.IsValid() || v.Kind() == dyn.KindNil {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}
			return v, nil
		}

		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
		resolved, err := dyn.MapByPattern(root, pattern, func(p dyn.Path, resource dyn.Value) (dyn.Value, error) {
			// A resolved id is a string; normalize so it gets the field's type (e.g. int job_id).
			v, err := dynvar.Resolve(resource, lookup)
			if err != nil {
				log.Debugf(ctx, "Not resolving references in %s for display: %s", p, err)
				return resource, nil
			}
			normalized, ok := normalizeResource(p[1].Key(), v)
			if !ok {
				log.Debugf(ctx, "Not resolving references in %s for display: resolved values do not match the resource type", p)
				return resource, nil
			}
			return normalized, nil
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Mutate leaves the typed config half-written if conversion fails, so check first.
		var check config.Root
		if err := convert.ToTyped(&check, resolved); err != nil {
			return dyn.InvalidValue, err
		}
		return resolved, nil
	})
	if err != nil {
		log.Debugf(ctx, "Not resolving resource references for display: %s", err)
	}
	return nil
}

// normalizeResource normalizes v, a resource of the given group, to the resource's Go type
// and reports whether it converts to that type without errors.
func normalizeResource(group string, v dyn.Value) (dyn.Value, bool) {
	typ, ok := config.ResourcesTypes()[group]
	if !ok {
		return dyn.InvalidValue, false
	}
	typed := reflect.New(typ).Interface()
	normalized, diags := convert.Normalize(typed, v)
	if len(diags) > 0 {
		return dyn.InvalidValue, false
	}
	if err := convert.ToTyped(typed, normalized); err != nil {
		return dyn.InvalidValue, false
	}
	return normalized, true
}
