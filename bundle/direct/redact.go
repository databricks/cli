package direct

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

const sensitiveRedactedValue = "[redacted]"

// sensitivePaths parses the sensitive field rules of a resource type.
func sensitivePaths(rules []dresources.FieldRule) ([]*structpath.PathNode, error) {
	var paths []*structpath.PathNode
	for _, rule := range rules {
		field := rule.Field.String()
		path, err := structpath.ParsePath(field)
		if err != nil {
			return nil, fmt.Errorf("parsing sensitive field path %q: %w", field, err)
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// redactSensitiveFields replaces (or zeros) scalar fields in s (a pointer to a typed struct)
// that the adapter marks as sensitive. replacement is what to set the field to; pass
// sensitiveRedactedValue for plan output display, or "" for state-file storage.
func redactSensitiveFields(adapter *dresources.Adapter, s any, replacement string) error {
	if s == nil {
		return nil
	}

	paths, err := sensitivePaths(adapter.ResourceConfig().SensitiveFields)
	if err != nil {
		return err
	}
	for _, path := range paths {
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

// redactPlanEntry redacts new_state, remote_state and changes of a plan entry. It runs once planning
// is done: reference resolution re-serializes new_state from the unredacted cache, and apply reads
// that cache, so the plan must not be serialized after this.
func redactPlanEntry(adapter *dresources.Adapter, entry *deployplan.PlanEntry) error {
	if len(adapter.GetSensitiveFields()) == 0 {
		return nil
	}

	if entry.NewState != nil {
		sv, err := entry.NewState.ToStructVar(adapter.StateType())
		if err != nil {
			return fmt.Errorf("redacting new_state: %w", err)
		}
		if err := redactStruct(adapter, sv.Value); err != nil {
			return fmt.Errorf("redacting new_state: %w", err)
		}
		redacted, err := sv.ToJSON()
		if err != nil {
			return fmt.Errorf("redacting new_state: %w", err)
		}
		*entry.NewState = *redacted
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

// sensitiveConfigValue returns the value the config sets for a sensitive field, which a deploy sends.
// The config value is used whenever it is set. An empty config value is an error only when the plan
// holds the field non-zero, since the plan then expects a value the config cannot supply. A field
// that is zero in both is output-only and yields "".
func sensitiveConfigValue(root *config.Root, resourceKey string, path *structpath.PathNode, planned any) (string, error) {
	rc, err := root.GetResourceConfig(resourceKey)
	if err != nil {
		return "", err
	}
	var value string
	if rc != nil {
		if v, err := structaccess.Get(rc, path); err == nil {
			value, _ = v.(string)
		}
	}
	if value != "" || planned == nil || planned == "" {
		return value, nil
	}
	if rc == nil {
		return "", fmt.Errorf("resource %s is not in the config; run 'bundle plan' again", resourceKey)
	}
	return "", fmt.Errorf("resource %s: the config has no value for sensitive field %s; set the variable it uses with --var or BUNDLE_VAR_<name>", resourceKey, path)
}

// restoreSensitiveFields replaces the sensitive values in s, a saved plan's state, with the values
// the current config sets. CheckSensitivePlanValues runs the same rule before the plan is approved.
func restoreSensitiveFields(adapter *dresources.Adapter, resourceKey string, s any, root *config.Root) error {
	paths, err := sensitivePaths(adapter.ResourceConfig().SensitiveFields)
	if err != nil {
		return err
	}
	for _, path := range paths {
		planned, err := structaccess.Get(s, path)
		if err != nil {
			continue
		}
		value, err := sensitiveConfigValue(root, resourceKey, path, planned)
		if err != nil {
			return err
		}
		if value == "" {
			continue
		}
		if err := structaccess.Set(s, path, value); err != nil {
			return fmt.Errorf("resource %s: restoring sensitive field %s: %w", resourceKey, path, err)
		}
	}
	return nil
}
