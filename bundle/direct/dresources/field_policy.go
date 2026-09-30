package dresources

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/libs/structs/structpath"
)

// FieldPolicyConfig is one resource's lifecycle config in the field-keyed format: a
// map from field prefix to its policy, plus a few resource-level settings. It is the
// authoring format for configs/<resource_type>.yml; Lower turns it into the flat
// ResourceLifecycleConfig the planner consumes.
//
// Keying by field makes a field's whole policy live in one place and makes a
// duplicate field a YAML load error, so the redundancy the flat format needed
// dedicated exclusivity/redundancy tests for is unrepresentable here.
type FieldPolicyConfig struct {
	// Action is a resource-root local-change action: the whole resource behaves this
	// way on any change (e.g. immutable when the API has no update endpoint). Mirrors
	// a flat action rule with the field omitted. Reason overrides its default.
	Action action `yaml:"action,omitempty"`
	Reason string `yaml:"reason,omitempty"`

	// RemoteAdditionsWhen gates ignore_remote_additions at the resource root (the
	// whole object is backend-co-owned when this field is set). Per-object gates go
	// on the field entry instead. See RemoteAdditionRule.
	RemoteAdditionsWhen *structpath.PathNode `yaml:"remote_additions_when,omitempty"`

	Fields map[string]FieldPolicy `yaml:"fields"`
}

// FieldPolicy is one field's lifecycle policy: a single Action (the local-change
// decision) plus independent modifiers on other axes (remote drift, comparison,
// backend defaults, storage, redaction, reference resolution). The modifiers never
// conflict with the Action, so a field cannot be handed contradictory behaviours.
type FieldPolicy struct {
	// Action is the local-change decision. Empty = a normal in-place update.
	Action action `yaml:"action,omitempty"`

	// Reason overrides the reason surfaced in the plan for Action. id/id_renameable/
	// immutable supply a default; ignore_local requires it (the justification varies).
	Reason string `yaml:"reason,omitempty"`

	// IgnoreRemote, when non-empty, skips remote-only drift on the field; its value is
	// the reason (which varies: managed, input_only, etag_based, spec:input_only, ...).
	IgnoreRemote string `yaml:"ignore_remote,omitempty"`

	// Compare customizes equality before the action is decided. "" | "trim_slash".
	Compare string `yaml:"compare,omitempty"`

	// BackendDefault skips a config-absent value the backend populated. BackendValues
	// constrains it to specific remote values and implies BackendDefault.
	BackendDefault bool  `yaml:"backend_default,omitempty"`
	BackendValues  []any `yaml:"backend_values,omitempty"`

	// Hashed persists the value to state as a content hash instead of the raw value.
	Hashed bool `yaml:"hashed,omitempty"`

	// Sensitive marks the field for redaction in logs and state.
	Sensitive bool `yaml:"sensitive,omitempty"`

	// StableOutput marks an output-only field the backend assigns at creation and never
	// changes, so a cross-resource reference resolves from the remote cache during a
	// keeps-ID update. See StableOutputFields in config.go.
	StableOutput bool `yaml:"stable_output,omitempty"`

	// RemoteAdditionsWhen gates ignore_remote_additions on this object (its contents are
	// backend-co-owned when the named sibling field is set).
	RemoteAdditionsWhen *structpath.PathNode `yaml:"remote_additions_when,omitempty"`
}

// action is the local-change decision for a field. Its UnmarshalYAML validates against
// validActions, so an unknown action is a parse-time error (with file/line).
type action string

const (
	actionMutable      action = ""              // default: a normal in-place update
	actionImmutable    action = "immutable"     // recreate on local change
	actionID           action = "id"            // provided id: recreate on change, skip remote drift
	actionIDRenameable action = "id_renameable" // updatable id: rename in place on change
	actionIgnoreLocal  action = "ignore_local"  // skip local edits
)

// validActions is the single source of truth for which actions parse. lowerAction must
// handle each; TestFieldPolicyEveryActionLowers keeps the two in sync.
var validActions = map[action]bool{
	actionMutable:      true,
	actionImmutable:    true,
	actionID:           true,
	actionIDRenameable: true,
	actionIgnoreLocal:  true,
}

func (a *action) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if !validActions[action(s)] {
		return fmt.Errorf("unknown action %q", s)
	}
	*a = action(s)
	return nil
}

const (
	compareTrimSlash = "trim_slash"

	// Reason strings the flat format spells out per entry; the field-keyed format
	// derives them from the action / modifier so the common case needs no reason.
	reasonIDField   = "id_field"
	reasonIDChanges = "id_changes"
	reasonImmutable = "immutable"
	reasonTrimSlash = "uc_strips_trailing_slash"
)

// Lower translates the field-keyed config into the flat ResourceLifecycleConfig the
// planner and every other consumer already use, so wiring in this format changes
// nothing downstream.
func (c FieldPolicyConfig) Lower() (ResourceLifecycleConfig, error) {
	var out ResourceLifecycleConfig

	// Resource-root rules (field omitted); mirror the flat loader's nil Field.
	if err := lowerAction(&out, nil, c.Action, c.Reason); err != nil {
		return out, err
	}
	if c.RemoteAdditionsWhen != nil {
		out.IgnoreRemoteAdditions = append(out.IgnoreRemoteAdditions, RemoteAdditionRule{Field: nil, WhenSet: c.RemoteAdditionsWhen})
	}

	// Deterministic order keeps the lowered lists stable for goldens and diffs.
	for _, field := range slices.Sorted(maps.Keys(c.Fields)) {
		p := c.Fields[field]
		pattern, err := structpath.ParsePattern(field)
		if err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}
		if err := lowerField(&out, pattern, field, p); err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}
	}

	return out, nil
}

// lowerAction appends the flat rule for one local-change action on pattern (nil = root).
func lowerAction(out *ResourceLifecycleConfig, pattern *structpath.PatternNode, act action, reason string) error {
	switch act {
	case actionMutable:
		if reason != "" {
			return fmt.Errorf("reason %q set without an action", reason)
		}
	case actionID:
		out.ProvidedIDFields = append(out.ProvidedIDFields, FieldRule{Field: pattern, Reason: reasonOr(reason, reasonIDField)})
	case actionIDRenameable:
		out.UpdatableIDFields = append(out.UpdatableIDFields, FieldRule{Field: pattern, Reason: reasonOr(reason, reasonIDChanges)})
	case actionImmutable:
		out.RecreateOnChanges = append(out.RecreateOnChanges, FieldRule{Field: pattern, Reason: reasonOr(reason, reasonImmutable)})
	case actionIgnoreLocal:
		out.IgnoreLocalChanges = append(out.IgnoreLocalChanges, FieldRule{Field: pattern, Reason: reason})
	default:
		// Unreachable via YAML (UnmarshalYAML validates); guards a value added to
		// validActions but not handled here.
		return fmt.Errorf("unhandled action %q", act)
	}
	return nil
}

func lowerField(out *ResourceLifecycleConfig, pattern *structpath.PatternNode, field string, p FieldPolicy) error {
	if err := lowerAction(out, pattern, p.Action, p.Reason); err != nil {
		return err
	}

	if p.IgnoreRemote != "" {
		out.IgnoreRemoteChanges = append(out.IgnoreRemoteChanges, FieldRule{Field: pattern, Reason: p.IgnoreRemote})
	}

	switch p.Compare {
	case "":
	case compareTrimSlash:
		out.NormalizeSlash = append(out.NormalizeSlash, FieldRule{Field: pattern, Reason: reasonTrimSlash})
	default:
		return fmt.Errorf("unknown compare %q", p.Compare)
	}

	if p.BackendDefault || len(p.BackendValues) > 0 {
		values, err := toRawValues(p.BackendValues)
		if err != nil {
			return err
		}
		out.BackendDefaults = append(out.BackendDefaults, BackendDefaultRule{Field: pattern, Values: values})
	}

	if p.Hashed {
		out.HashedFields = append(out.HashedFields, field)
	}
	if p.Sensitive {
		out.SensitiveFields = append(out.SensitiveFields, FieldRule{Field: pattern, Reason: ""})
	}
	if p.StableOutput {
		out.StableOutputFields = append(out.StableOutputFields, FieldRule{Field: pattern, Reason: ""})
	}
	if p.RemoteAdditionsWhen != nil {
		out.IgnoreRemoteAdditions = append(out.IgnoreRemoteAdditions, RemoteAdditionRule{Field: pattern, WhenSet: p.RemoteAdditionsWhen})
	}
	return nil
}

func reasonOr(reason, def string) string {
	if reason == "" {
		return def
	}
	return reason
}

// toRawValues marshals native YAML backend_values to JSON, matching BackendDefaultRule.UnmarshalYAML.
func toRawValues(vals []any) ([]json.RawMessage, error) {
	if len(vals) == 0 {
		return nil, nil
	}
	out := make([]json.RawMessage, 0, len(vals))
	for _, v := range vals {
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}
