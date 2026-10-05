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

// FieldPolicy is one field's lifecycle policy. It is authored as a list of behaviour
// tokens (the common case) or, when a behaviour needs a value, as a map:
//
//	volume_type:  [immutable]
//	storage_root: [immutable, trim_slash]
//	create_time:  [mutable_output]
//	name:         [immutable_output]
//	value:        { ignore_remote: etag_based, hashed: true }
//	replace_existing: { ignore_local: input_only }
//
// A field carries at most one local-change action (id / renameable_id / immutable /
// ignore_local); the rest are independent modifiers, so nothing contradicts.
type FieldPolicy struct {
	// Action is the local-change decision (from an id/renameable_id/immutable token, or
	// the ignore_local map key). Empty means a normal in-place update.
	Action action

	// Reason is the plan reason for ignore_local (the only action whose reason is not
	// derived); other actions and the modifiers derive theirs.
	Reason string

	// IgnoreRemote, when non-empty, skips remote-only drift; its value is the reason.
	// From the input_only / mutable_output tokens (reason input_only / output_only,
	// which Lower prefixes with "spec:" in generated configs) or the ignore_remote map
	// key (a custom reason: managed, etag_based, auto, ...).
	IgnoreRemote string

	// StableOutput marks a backend-assigned output that never changes (the
	// immutable_output token): reference-safe, so ${resources.X.<field>} resolves from
	// the remote cache during an in-place update. See StableOutputFields in config.go.
	StableOutput bool

	Compare             string // trim_slash token -> normalize_slash
	Hashed              bool
	Sensitive           bool
	BackendDefault      bool
	BackendValues       []any
	RemoteAdditionsWhen *structpath.PathNode
}

// Behaviour tokens (list form) and the map keys they mirror.
const (
	tokenID              = "id"
	tokenRenameableID    = "renameable_id"
	tokenImmutable       = "immutable"
	tokenInputOnly       = "input_only"
	tokenMutableOutput   = "mutable_output"
	tokenImmutableOutput = "immutable_output"
	tokenTrimSlash       = "trim_slash"
	tokenHashed          = "hashed"
	tokenSensitive       = "sensitive"
	tokenBackendDefault  = "backend_default"

	// Base reasons the tokens imply; Lower prefixes "spec:" in generated configs.
	reasonIDField    = "id_field"
	reasonIDChanges  = "id_changes"
	reasonImmutable  = "immutable"
	reasonInputOnly  = "input_only"
	reasonOutputOnly = "output_only"
	reasonTrimSlash  = "uc_strips_trailing_slash"
	compareTrimSlash = "trim_slash"
)

func (p *FieldPolicy) applyToken(t string) error {
	switch t {
	case tokenID:
		p.Action = actionID
	case tokenRenameableID:
		p.Action = actionRenameableID
	case tokenImmutable:
		p.Action = actionImmutable
	case tokenInputOnly:
		p.IgnoreRemote = reasonInputOnly
	case tokenMutableOutput:
		p.IgnoreRemote = reasonOutputOnly
	case tokenImmutableOutput:
		p.StableOutput = true
	case tokenTrimSlash:
		p.Compare = compareTrimSlash
	case tokenHashed:
		p.Hashed = true
	case tokenSensitive:
		p.Sensitive = true
	case tokenBackendDefault:
		p.BackendDefault = true
	default:
		return fmt.Errorf("unknown behaviour %q", t)
	}
	return nil
}

// UnmarshalYAML accepts the list form (a sequence of tokens) or the map form (keys that
// carry a value). A duplicate field key is already a YAML error at the map above.
func (p *FieldPolicy) UnmarshalYAML(unmarshal func(any) error) error {
	var tokens []string
	if err := unmarshal(&tokens); err == nil {
		for _, t := range tokens {
			if err := p.applyToken(t); err != nil {
				return err
			}
		}
		return nil
	}

	var m struct {
		Action              action               `yaml:"action"`
		Reason              string               `yaml:"reason"`
		IgnoreRemote        string               `yaml:"ignore_remote"`
		IgnoreLocal         string               `yaml:"ignore_local"`
		Compare             string               `yaml:"compare"`
		Hashed              bool                 `yaml:"hashed"`
		Sensitive           bool                 `yaml:"sensitive"`
		StableOutput        bool                 `yaml:"stable_output"`
		BackendDefault      bool                 `yaml:"backend_default"`
		BackendValues       []any                `yaml:"backend_values"`
		RemoteAdditionsWhen *structpath.PathNode `yaml:"remote_additions_when"`
	}
	if err := unmarshal(&m); err != nil {
		return err
	}
	p.Action = m.Action
	p.Reason = m.Reason
	p.IgnoreRemote = m.IgnoreRemote
	p.Compare = m.Compare
	p.Hashed = m.Hashed
	p.Sensitive = m.Sensitive
	p.StableOutput = m.StableOutput
	p.BackendDefault = m.BackendDefault
	p.BackendValues = m.BackendValues
	p.RemoteAdditionsWhen = m.RemoteAdditionsWhen
	if m.IgnoreLocal != "" {
		p.Action = actionIgnoreLocal
		p.Reason = m.IgnoreLocal
	}
	return nil
}

// action is the local-change decision for a field.
type action string

const (
	actionMutable      action = ""              // default: a normal in-place update
	actionImmutable    action = "immutable"     // recreate on local change
	actionID           action = "id"            // provided id: recreate on change, skip remote drift
	actionRenameableID action = "renameable_id" // updatable id: rename in place on change
	actionIgnoreLocal  action = "ignore_local"  // skip local edits
)

var validActions = map[action]bool{
	actionMutable:      true,
	actionImmutable:    true,
	actionID:           true,
	actionRenameableID: true,
	actionIgnoreLocal:  true,
}

// UnmarshalYAML validates the resource-root action (map key form); field tokens are
// validated in applyToken.
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

// Lower translates the field-keyed config into the flat ResourceLifecycleConfig the
// planner and every other consumer already use. generated marks a *.generated.yml file,
// whose reasons are all spec-derived and so carry the "spec:" prefix.
func (c FieldPolicyConfig) Lower(generated bool) (ResourceLifecycleConfig, error) {
	var out ResourceLifecycleConfig

	// Resource-root rules (field omitted); mirror the flat loader's nil Field.
	if err := lowerAction(&out, nil, c.Action, c.Reason, generated); err != nil {
		return out, err
	}
	if c.RemoteAdditionsWhen != nil {
		out.IgnoreRemoteAdditions = append(out.IgnoreRemoteAdditions, RemoteAdditionRule{Field: nil, WhenSet: c.RemoteAdditionsWhen})
	}

	for _, field := range slices.Sorted(maps.Keys(c.Fields)) {
		p := c.Fields[field]
		pattern, err := structpath.ParsePattern(field)
		if err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}
		if err := lowerField(&out, pattern, field, p, generated); err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}
	}

	return out, nil
}

func specReason(base string, generated bool) string {
	if generated {
		return "spec:" + base
	}
	return base
}

// lowerAction appends the flat rule for one local-change action on pattern (nil = root).
func lowerAction(out *ResourceLifecycleConfig, pattern *structpath.PatternNode, act action, reason string, generated bool) error {
	switch act {
	case actionMutable:
		if reason != "" {
			return fmt.Errorf("reason %q set without an action", reason)
		}
	case actionID:
		out.ProvidedIDFields = append(out.ProvidedIDFields, FieldRule{Field: pattern, Reason: reasonOr(reason, reasonIDField)})
	case actionRenameableID:
		out.UpdatableIDFields = append(out.UpdatableIDFields, FieldRule{Field: pattern, Reason: reasonOr(reason, reasonIDChanges)})
	case actionImmutable:
		out.RecreateOnChanges = append(out.RecreateOnChanges, FieldRule{Field: pattern, Reason: reasonOr(reason, specReason(reasonImmutable, generated))})
	case actionIgnoreLocal:
		out.IgnoreLocalChanges = append(out.IgnoreLocalChanges, FieldRule{Field: pattern, Reason: reason})
	default:
		// Unreachable: UnmarshalYAML validates actions against validActions.
		return fmt.Errorf("unhandled action %q", act)
	}
	return nil
}

func lowerField(out *ResourceLifecycleConfig, pattern *structpath.PatternNode, field string, p FieldPolicy, generated bool) error {
	if err := lowerAction(out, pattern, p.Action, p.Reason, generated); err != nil {
		return err
	}

	if p.IgnoreRemote != "" {
		out.IgnoreRemoteChanges = append(out.IgnoreRemoteChanges, FieldRule{Field: pattern, Reason: specReason(p.IgnoreRemote, generated)})
	}
	if p.StableOutput {
		out.StableOutputFields = append(out.StableOutputFields, FieldRule{Field: pattern, Reason: ""})
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
