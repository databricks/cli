package direct

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/databricks/cli/libs/structs/structvar"
)

const sensitiveRedactedValue = "[redacted]"

// redactStruct zeros all scalar fields in s (a pointer to a typed struct) that
// the adapter marks as sensitive. The struct is modified in place.
func redactStruct(adapter *dresources.Adapter, s any) error {
	if s == nil {
		return nil
	}

	var toRedact []*structpath.PathNode
	err := structwalk.Walk(s, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
		if adapter.IsSensitive(path) {
			toRedact = append(toRedact, path)
		}
	})
	if err != nil {
		return fmt.Errorf("walking struct: %w", err)
	}

	for _, path := range toRedact {
		if err := structaccess.Set(s, path, sensitiveRedactedValue); err != nil {
			// Field might not be a string (e.g. it could be an int or bool); try
			// setting it to its zero value instead.
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

// redactChanges replaces the Old, New, and Remote values in any ChangeDesc whose
// path is marked sensitive by the adapter, so the plan output does not leak them.
func redactChanges(adapter *dresources.Adapter, changes deployplan.Changes) error {
	for pathString, ch := range changes {
		path, err := structpath.ParsePath(pathString)
		if err != nil {
			return fmt.Errorf("parsing change path %q: %w", pathString, err)
		}
		if adapter.IsSensitive(path) {
			if ch.Old != nil {
				ch.Old = sensitiveRedactedValue
			}
			if ch.New != nil {
				ch.New = sensitiveRedactedValue
			}
			if ch.Remote != nil {
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
