package dresources

import (
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/structs/structpath"
	"go.yaml.in/yaml/v3"
)

// FieldPolicy is the field-keyed alternative to the flat per-behaviour lists in
// ResourceLifecycleConfig. A field prefix names exactly one Action plus optional
// orthogonal modifiers, so a field cannot be handed two contradictory actions:
// the config is a map keyed by field, and the YAML loader rejects a duplicate key.
// That makes the action-exclusivity checks in config_test.go unnecessary — the
// redundancy is unrepresentable rather than merely tested against.
//
// Action decides what happens when the field differs. The three modifiers
// (Compare, Hashed, Sensitive) live on independent axes — comparison, state
// storage, redaction — so they legitimately combine with any Action and can
// never contradict it.
type FieldPolicy struct {
	// Action is the single lifecycle decision for the field. Empty means the
	// default (a normal in-place update). Its UnmarshalYAML rejects any value
	// outside validActions, so an unknown action fails at parse time.
	Action action `yaml:"action,omitempty"`

	// Reason overrides the default reason string surfaced in the plan. Actions
	// with an obvious reason (id, immutable, ...) supply a default; ignore*
	// actions require it because the justification is resource-specific.
	Reason string `yaml:"reason,omitempty"`

	// Compare customizes equality before the action is decided. A string (not a
	// bool) because comparison modes can grow and could combine — "" | "trim_slash"
	// today, "ci" etc. later.
	Compare string `yaml:"compare,omitempty"`

	// Hashed persists the value to state as a content hash instead of the raw
	// value. For large, equality-only fields never read back from state.
	Hashed bool `yaml:"hashed,omitempty"`

	// Sensitive marks the field for redaction in logs and state.
	Sensitive bool `yaml:"sensitive,omitempty"`

	// StableOutput marks an output-only field the backend assigns at creation and
	// never changes, so a cross-resource reference to it resolves from the remote
	// cache during a keeps-ID update instead of being delayed. See
	// StableOutputFields in config.go.
	StableOutput bool `yaml:"stable_output,omitempty"`
}

// action is the lifecycle decision for a field. It is a defined type with an
// UnmarshalYAML that validates against validActions, so an unknown action is a
// parse-time error (with file/line) rather than a failure discovered in Lower.
type action string

// Action values. Each maps to one action rung of the plan ladder; see the table
// in README.md. immutable/id/id_renameable all recreate-or-rename on a local
// change and skip a remote-only diff; ignore* skip; backend_owned tolerates a
// value the backend populated.
const (
	actionMutable      action = ""              // default: a normal in-place update
	actionImmutable    action = "immutable"     // recreate on local change
	actionID           action = "id"            // provided id: recreate on change, skip remote drift
	actionIDRenameable action = "id_renameable" // updatable id: rename in place on change
	actionIgnore       action = "ignore"        // skip both local edits and remote drift
	actionIgnoreLocal  action = "ignore_local"  // skip local edits only
	actionIgnoreRemote action = "ignore_remote" // skip remote drift only
	actionBackendOwned action = "backend_owned" // skip a config-absent value the backend set
)

// validActions is the single source of truth for which actions parse. The
// lowerAction switch must handle each; TestFieldPolicyEveryActionLowers keeps the
// two in sync.
var validActions = map[action]bool{
	actionMutable:      true,
	actionImmutable:    true,
	actionID:           true,
	actionIDRenameable: true,
	actionIgnore:       true,
	actionIgnoreLocal:  true,
	actionIgnoreRemote: true,
	actionBackendOwned: true,
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

// FieldPolicyConfig is one resource's field-keyed lifecycle config.
type FieldPolicyConfig struct {
	Fields map[string]FieldPolicy `yaml:"fields"`
}

// Lower translates the field-keyed config into the existing ResourceLifecycleConfig
// so the plan ladder and every consumer keep working unchanged. It is the seam that
// lets the two formats coexist during migration.
func (c FieldPolicyConfig) Lower() (ResourceLifecycleConfig, error) {
	var out ResourceLifecycleConfig

	// Deterministic order keeps the lowered lists stable for goldens and diffs.
	for _, field := range slices.Sorted(maps.Keys(c.Fields)) {
		p := c.Fields[field]
		pattern, err := structpath.ParsePattern(field)
		if err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}

		if err := lowerAction(&out, pattern, p); err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}
		if err := lowerModifiers(&out, pattern, field, p); err != nil {
			return out, fmt.Errorf("field %q: %w", field, err)
		}
	}

	return out, nil
}

func lowerAction(out *ResourceLifecycleConfig, pattern *structpath.PatternNode, p FieldPolicy) error {
	rule := FieldRule{Field: pattern, Reason: p.Reason}
	switch p.Action {
	case actionMutable:
		if p.Reason != "" {
			return fmt.Errorf("reason %q set without an action", p.Reason)
		}
	case actionID:
		out.ProvidedIDFields = append(out.ProvidedIDFields, withDefaultReason(rule, reasonIDField))
	case actionIDRenameable:
		out.UpdatableIDFields = append(out.UpdatableIDFields, withDefaultReason(rule, reasonIDChanges))
	case actionImmutable:
		out.RecreateOnChanges = append(out.RecreateOnChanges, withDefaultReason(rule, reasonImmutable))
	case actionIgnore:
		out.IgnoreLocalChanges = append(out.IgnoreLocalChanges, rule)
		out.IgnoreRemoteChanges = append(out.IgnoreRemoteChanges, rule)
	case actionIgnoreLocal:
		out.IgnoreLocalChanges = append(out.IgnoreLocalChanges, rule)
	case actionIgnoreRemote:
		out.IgnoreRemoteChanges = append(out.IgnoreRemoteChanges, rule)
	case actionBackendOwned:
		out.BackendDefaults = append(out.BackendDefaults, BackendDefaultRule{Field: pattern})
	default:
		// Unreachable via YAML (UnmarshalYAML validates); guards a value added to
		// validActions but not handled here.
		return fmt.Errorf("unhandled action %q", p.Action)
	}
	return nil
}

func lowerModifiers(out *ResourceLifecycleConfig, pattern *structpath.PatternNode, field string, p FieldPolicy) error {
	switch p.Compare {
	case "":
	case compareTrimSlash:
		out.NormalizeSlash = append(out.NormalizeSlash, FieldRule{Field: pattern, Reason: reasonTrimSlash})
	default:
		return fmt.Errorf("unknown compare %q", p.Compare)
	}

	if p.Hashed {
		out.HashedFields = append(out.HashedFields, field)
	}

	if p.Sensitive {
		out.SensitiveFields = append(out.SensitiveFields, FieldRule{Field: pattern})
	}

	if p.StableOutput {
		out.StableOutputFields = append(out.StableOutputFields, FieldRule{Field: pattern})
	}
	return nil
}

func withDefaultReason(rule FieldRule, def string) FieldRule {
	if rule.Reason == "" {
		rule.Reason = def
	}
	return rule
}

//go:embed configs_v2/*.yml
var fieldPolicyFS embed.FS

const configV2Dir = "configs_v2"

// loadFieldPolicyConfigs parses every configs_v2/<resource_type>.yml and lowers it
// to a ResourceLifecycleConfig, keyed by resource type.
var loadFieldPolicyConfigs = sync.OnceValue(func() map[string]ResourceLifecycleConfig {
	names, err := fs.Glob(fieldPolicyFS, configV2Dir+"/*"+ymlSuffix)
	if err != nil {
		panic(err)
	}

	result := map[string]ResourceLifecycleConfig{}
	for _, name := range names {
		resourceType := strings.TrimSuffix(path.Base(name), ymlSuffix)

		data, err := fieldPolicyFS.ReadFile(name)
		if err != nil {
			panic(err)
		}

		var fpc FieldPolicyConfig
		if err := yaml.Unmarshal(data, &fpc); err != nil {
			panic(fmt.Errorf("%s: %w", name, err))
		}

		lowered, err := fpc.Lower()
		if err != nil {
			panic(fmt.Errorf("%s: %w", name, err))
		}
		result[resourceType] = lowered
	}
	return result
})

// GetFieldPolicyConfig returns the lowered lifecycle config authored in the
// field-keyed configs_v2/ format, or nil if the resource has no such file.
func GetFieldPolicyConfig(resourceType string) *ResourceLifecycleConfig {
	if rc, ok := loadFieldPolicyConfigs()[resourceType]; ok {
		return &rc
	}
	return nil
}
