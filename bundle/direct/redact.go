package direct

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

const sensitiveRedactedValue = "[redacted]"

// redactSensitiveFields replaces (or zeros) scalar fields in s (a pointer to a typed struct)
// that the adapter marks as sensitive. replacement is what to set the field to; pass
// sensitiveRedactedValue for plan output display, or "" for state-file storage.
func redactSensitiveFields(adapter *dresources.Adapter, s any, replacement string) error {
	if s == nil {
		return nil
	}

	fields := adapter.GetSensitiveFields()
	for _, field := range fields {
		path, err := structpath.ParsePath(field)
		if err != nil {
			return fmt.Errorf("parsing sensitive field path %q: %w", field, err)
		}

		fv, err := structaccess.Get(s, path)
		if err != nil {
			continue
		}
		if fv == nil {
			// structaccess.Get returns nil for omitempty fields whose value is zero.
			// Such fields are absent from JSON, so there is nothing to redact or zero.
			continue
		}
		_ = structaccess.Set(s, path, replacement)
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
	fields := adapter.GetSensitiveFields()
	for pathString, ch := range changes {
		if slices.Contains(fields, pathString) {
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

// redactPlanEntry redacts sensitive fields from remote_state and changes of a plan
// entry. new_state is already redacted before serialization in makePlan.
func redactPlanEntry(adapter *dresources.Adapter, entry *deployplan.PlanEntry) error {
	if len(adapter.GetSensitiveFields()) == 0 {
		return nil
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
