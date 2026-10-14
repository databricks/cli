package resourcemutator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// Suffix matching for_each_task.inputs at any nesting depth within a job.
var forEachTaskInputsSuffix = dyn.NewPath(dyn.Key("for_each_task"), dyn.Key("inputs"))

var varShorthandPrefix = dyn.NewPath(dyn.Key("var"))

type serializeForEachTaskInputs struct{}

// SerializeForEachTaskInputs converts a complex variable referenced by
// for_each_task.inputs into the JSON string the Jobs API expects.
//
// The inputs field is typed as a string, so a substituted sequence or map
// would be dropped by normalization with an "expected string, found sequence"
// warning. Serializing here, before variable resolution runs, keeps the value
// intact. Anything that is not a pure reference to a complex variable value
// is left untouched for the regular resolution and validation to handle.
func SerializeForEachTaskInputs() bundle.Mutator {
	return &serializeForEachTaskInputs{}
}

func (s serializeForEachTaskInputs) Name() string {
	return "SerializeForEachTaskInputs"
}

func (s serializeForEachTaskInputs) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
	)

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return dyn.MapByPattern(root, pattern, func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
			return dyn.Walk(job, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				if !p.HasSuffix(forEachTaskInputsSuffix) {
					return v, nil
				}
				return serializeInputsValue(root, v)
			})
		})
	})

	diags = diags.Extend(diag.FromErr(err))
	return diags
}

func serializeInputsValue(root, v dyn.Value) (dyn.Value, error) {
	s, ok := v.AsString()
	if !ok {
		return v, nil
	}

	path, ok := dynvar.PureReferenceToPath(s)
	if !ok {
		return v, nil
	}

	path, ok = variableValuePath(path)
	if !ok {
		return v, nil
	}

	target, ok := lookupVariableValue(root, path)
	if !ok {
		return v, nil
	}

	switch target.Kind() {
	case dyn.KindSequence, dyn.KindMap:
	default:
		return v, nil
	}

	// Only serialize fully resolved values. Baking an unresolved reference
	// into JSON would change its meaning from structural substitution to
	// string interpolation when a later resolution round picks it up.
	if containsVariableReference(target) {
		return v, nil
	}

	jsonBytes, err := json.Marshal(target.AsAny())
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to serialize for_each_task.inputs: %w", err)
	}

	return dyn.NewValue(string(jsonBytes), v.Locations()), nil
}

// variableValuePath maps a pure reference to the variable value it points to.
// ${var.foo} is shorthand for ${variables.foo.value}; longer paths address
// into the value itself. References to anything else are not variable values.
func variableValuePath(path dyn.Path) (dyn.Path, bool) {
	if path.HasPrefix(varShorthandPrefix) {
		if len(path) < 2 {
			return nil, false
		}
		out := dyn.NewPath(dyn.Key("variables"), path[1], dyn.Key("value"))
		if len(path) > 2 {
			out = out.Append(path[2:]...)
		}
		return out, true
	}

	if len(path) > 0 && path[0].Key() == "variables" {
		return path, true
	}

	return nil, false
}

// lookupVariableValue resolves a variable value path against the current
// configuration. A missing reference is not an error here; the regular
// variable resolution reports it with suggestions.
func lookupVariableValue(root dyn.Value, path dyn.Path) (dyn.Value, bool) {
	target, err := dyn.GetByPath(root, path)
	if err != nil || !target.IsValid() {
		return dyn.InvalidValue, false
	}
	return target, true
}

func containsVariableReference(v dyn.Value) bool {
	found := false
	_ = dyn.WalkReadOnly(v, func(_ dyn.Path, v dyn.Value) error {
		if s, ok := v.AsString(); ok && dynvar.ContainsVariableReference(s) {
			found = true
		}
		return nil
	})
	return found
}
