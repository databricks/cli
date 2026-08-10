package direct

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/cli/libs/structs/structwalk"
)

const sensitiveRedactedValue = "[redacted]"

// redactSensitiveFields replaces (or zeros) scalar fields in s (a pointer to a typed struct)
// that the adapter marks as sensitive. replacement is what to set the field to; pass
// sensitiveRedactedValue for plan output display, or "" for state-file storage.
func redactSensitiveFields(adapter *dresources.Adapter, s any, replacement string) error {
	if s == nil {
		return nil
	}

	var toRedact []*structpath.PathNode
	err := structwalk.Walk(s, func(path *structpath.PathNode, val any, _ *reflect.StructField) {
		if adapter.IsSensitive(path) {
			toRedact = append(toRedact, path)
		}
	})
	if err != nil {
		return fmt.Errorf("walking struct: %w", err)
	}

	for _, path := range toRedact {
		if err := structaccess.Set(s, path, replacement); err != nil {
			// Field is not a string; fall back to its zero value.
			fv, ferr := structaccess.Get(s, path)
			if ferr != nil {
				continue
			}
			zero := reflect.Zero(reflect.TypeOf(fv)).Interface()
			_ = structaccess.Set(s, path, zero)
		}
	}

	return nil
}

// redactStruct replaces sensitive fields with "[redacted]" for display in plan output.
func redactStruct(adapter *dresources.Adapter, s any) error {
	return redactSensitiveFields(adapter, s, sensitiveRedactedValue)
}

// zeroSensitiveFields clears sensitive fields to their zero value for safe storage in
// the state file. This avoids a false "local change" diff on the next plan (the state
// stores "" instead of the actual value, so old==new==nil/zero and drift detection
// falls back to remote comparison via RemoteAlreadySet).
func zeroSensitiveFields(adapter *dresources.Adapter, s any) error {
	return redactSensitiveFields(adapter, s, "")
}

// redactChanges replaces the Old, New, and Remote values in any ChangeDesc whose
// path is marked sensitive by the adapter, so the plan output does not leak them.
// Empty/nil values are left as-is (they carry no secret).
func redactChanges(adapter *dresources.Adapter, changes deployplan.Changes) error {
	for pathString, ch := range changes {
		path, err := structpath.ParsePath(pathString)
		if err != nil {
			return fmt.Errorf("parsing change path %q: %w", pathString, err)
		}
		if adapter.IsSensitive(path) {
			if v, ok := ch.Old.(string); ok && v != "" {
				ch.Old = sensitiveRedactedValue
			}
			if v, ok := ch.New.(string); ok && v != "" {
				ch.New = sensitiveRedactedValue
			}
			if v, ok := ch.Remote.(string); ok && v != "" {
				ch.Remote = sensitiveRedactedValue
			}
		}
	}
	return nil
}

// redactNewStateJSON redacts sensitive fields inside a StructVarJSON by round-tripping
// through the adapter's state type so path matching works correctly.
func redactNewStateJSON(adapter *dresources.Adapter, svj *structvar.StructVarJSON) error {
	if svj == nil || len(svj.Value) == 0 {
		return nil
	}

	stateType := adapter.StateType()
	// StateType returns a pointer type; Elem gives the concrete struct type.
	ptr := reflect.New(stateType.Elem()).Interface()
	if err := json.Unmarshal(svj.Value, ptr); err != nil {
		return fmt.Errorf("unmarshaling new_state: %w", err)
	}

	if err := redactStruct(adapter, ptr); err != nil {
		return fmt.Errorf("redacting new_state: %w", err)
	}

	redacted, err := json.Marshal(ptr)
	if err != nil {
		return fmt.Errorf("re-marshaling new_state: %w", err)
	}
	svj.Value = redacted
	return nil
}

// redactPlanEntry redacts sensitive fields from all three sections of a plan entry:
// new_state, remote_state, and changes. The entry is modified in place.
func redactPlanEntry(adapter *dresources.Adapter, entry *deployplan.PlanEntry) error {
	if entry.NewState != nil {
		if err := redactNewStateJSON(adapter, entry.NewState); err != nil {
			return err
		}
	}

	if entry.RemoteState != nil {
		if err := redactStruct(adapter, entry.RemoteState); err != nil {
			return fmt.Errorf("redacting remote_state: %w", err)
		}
	}

	if entry.Changes != nil {
		if err := redactChanges(adapter, entry.Changes); err != nil {
			return fmt.Errorf("redacting changes: %w", err)
		}
	}

	return nil
}
