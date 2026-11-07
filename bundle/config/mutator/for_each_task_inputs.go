package mutator

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// forEachTaskInputsSuffix matches for_each_task.inputs at any nesting depth.
var forEachTaskInputsSuffix = dyn.NewPath(dyn.Key("for_each_task"), dyn.Key("inputs"))

// serializeForEachTaskInputs converts complex values substituted into
// for_each_task.inputs into the JSON string the Jobs API expects.
//
// The inputs field is typed as a string, so a substituted sequence or map
// would be dropped by normalization with an "expected string, found sequence"
// warning. Serializing here, after substitution and before normalization,
// keeps the value intact. This covers both direct references
// (inputs: ${var.table_list}) and references nested inside variable-defined
// structures (tasks: ${var.job_tasks}), which only materialize once the
// enclosing reference is substituted.
//
// Values still containing references are left untouched. Baking an unresolved
// reference into JSON would change its meaning from structural substitution
// to string interpolation, which coerces non-string values to text and can
// corrupt the JSON when the result contains quotes. Normalization then
// reports such values as before.
func serializeForEachTaskInputs(root dyn.Value) (dyn.Value, error) {
	pattern := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
	)

	return dyn.MapByPattern(root, pattern, func(_ dyn.Path, job dyn.Value) (dyn.Value, error) {
		return dyn.Walk(job, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if !p.HasSuffix(forEachTaskInputsSuffix) {
				return v, nil
			}
			return serializeInputsValue(v)
		})
	})
}

func serializeInputsValue(v dyn.Value) (dyn.Value, error) {
	switch v.Kind() {
	case dyn.KindSequence, dyn.KindMap:
	default:
		return v, nil
	}

	if containsVariableReference(v) {
		return v, nil
	}

	jsonBytes, err := json.Marshal(v.AsAny())
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to serialize for_each_task.inputs: %w", err)
	}

	return dyn.NewValue(string(jsonBytes), v.Locations()), nil
}

// containsVariableReference reports whether any string in the value,
// including map keys, holds an unescaped variable reference. Map keys are
// never resolved elsewhere, so a key holding a reference must keep the value
// out of serialization: the marshaled JSON would otherwise expose the key to
// interpolation by later resolution rounds.
func containsVariableReference(v dyn.Value) bool {
	found := false
	_, _ = dyn.Walk(v, func(_ dyn.Path, n dyn.Value) (dyn.Value, error) {
		if s, ok := n.AsString(); ok && dynvar.ContainsVariableReference(s) {
			found = true
		}
		if n.Kind() == dyn.KindMap {
			for _, pair := range n.MustMap().Pairs() {
				if s, ok := pair.Key.AsString(); ok && dynvar.ContainsVariableReference(s) {
					found = true
				}
			}
		}
		return n, nil
	})
	return found
}
