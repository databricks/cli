package direct

import (
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
// is done: the plan must not be serialized before this, and apply reads the unredacted cache, not
// new_state, so nothing after this point re-serializes the values into the entry.
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

// restoreSensitiveFields restores the sensitive fields of a saved plan's state, planned, from the
// config. The values are built the way makePlan builds them, so a deploy from the plan sends what a
// plain deploy sends: a reference stays in the refs and is resolved at apply, and an escaped "$${...}"
// is sent as the literal "${...}". A field the plan holds non-zero must have a config value.
func restoreSensitiveFields(adapter *dresources.Adapter, resourceKey string, planned *structvar.StructVar, root *config.Root) error {
	rules := dresources.GetResourceConfig(config.GetResourceTypeFromKey(resourceKey)).SensitiveFields
	if len(rules) == 0 {
		return nil
	}
	paths, err := sensitivePaths(rules)
	if err != nil {
		return err
	}

	// Each resource is looked up once, so a missing one is reported once rather than once per field.
	inputConfig, err := root.GetResourceConfig(resourceKey)
	if err != nil {
		return err
	}
	if inputConfig == nil {
		for _, path := range paths {
			if v, err := structaccess.Get(planned.Value, path); err == nil && !isZeroValue(v) {
				return fmt.Errorf("resource %s is not in the config; run 'bundle plan' again", resourceKey)
			}
		}
		return nil
	}

	inputStructVar, err := adapter.PrepareInputConfig(inputConfig, resourceKey)
	if err != nil {
		return fmt.Errorf("resource %s: %w", resourceKey, err)
	}
	built, err := adapter.PrepareState(inputStructVar.Value)
	if err != nil {
		return fmt.Errorf("resource %s: %w", resourceKey, err)
	}

	// The references are the same ones makePlan records for the resource.
	refs, err := extractReferences(root.Value(), resourceKey, adapter.StateType())
	if err != nil {
		return fmt.Errorf("resource %s: reading references: %w", resourceKey, err)
	}
	maps.Copy(refs, inputStructVar.Refs)

	// Checked before unescaping: an escaped "$${...}" is a literal, but an unescaped reference
	// without a recorded ref would be sent as it is.
	for _, path := range paths {
		raw, err := structaccess.Get(built, path)
		if err != nil {
			continue
		}
		if s, _ := raw.(string); dynvar.ContainsVariableReference(s) && refs[path.String()] == "" {
			return fmt.Errorf("resource %s: sensitive field %s holds an unresolved reference %q", resourceKey, path, s)
		}
	}

	if err := unescapeRefs(built); err != nil {
		return fmt.Errorf("resource %s: %w", resourceKey, err)
	}

	for _, path := range paths {
		v, err := structaccess.Get(built, path)
		if err != nil {
			continue
		}
		value, _ := v.(string)
		plannedValue, err := structaccess.Get(planned.Value, path)
		if err != nil {
			continue
		}
		if value == "" {
			if !isZeroValue(plannedValue) {
				return fmt.Errorf("resource %s: the config has no value for sensitive field %s; set the variable it uses with --var or BUNDLE_VAR_<name>", resourceKey, path)
			}
			continue
		}
		if err := structaccess.Set(planned.Value, path, value); err != nil {
			return fmt.Errorf("resource %s: restoring sensitive field %s: %w", resourceKey, path, err)
		}
		if ref, ok := refs[path.String()]; ok {
			if planned.Refs == nil {
				planned.Refs = make(map[string]string)
			}
			planned.Refs[path.String()] = ref
		}
	}
	return nil
}

func isZeroValue(v any) bool {
	return v == nil || v == ""
}
