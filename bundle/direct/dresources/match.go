package dresources

import (
	"encoding/json"
	"reflect"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
)

// FindMatchingRule returns the reason of the first rule whose field pattern
// is a prefix of the given path.
func FindMatchingRule(path *structpath.PathNode, rules []FieldRule) (string, bool) {
	for _, r := range rules {
		if path.HasPatternPrefix(r.Field) {
			return r.Reason, true
		}
	}
	return "", false
}

// MatchesBackendDefault reports whether the remote value at path matches one of
// the backend_defaults rules. If a rule has allowed values, the remote value
// must match one of them.
func (cfg *ResourceLifecycleConfig) MatchesBackendDefault(path *structpath.PathNode, remote any) bool {
	if cfg == nil || remote == nil {
		return false
	}
	if cfg.matchesBackendDefaultRule(path, remote) {
		return true
	}

	// Nil-vs-map case from structdiff: a remote-only map change is emitted at the
	// parent path rather than per key. Only skip the parent map if every remote
	// entry matches a configured backend-default child rule; any unmanaged key
	// must still surface as drift. rv is always valid here (remote != nil
	// above) and a nil map is excluded by Len() == 0.
	rv := reflect.ValueOf(remote)
	if rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String || rv.Len() == 0 {
		return false
	}
	iter := rv.MapRange()
	for iter.Next() {
		childPath := structpath.NewBracketString(path, iter.Key().String())
		if !cfg.matchesBackendDefaultRule(childPath, iter.Value().Interface()) {
			return false
		}
	}
	return true
}

// matchesBackendDefaultRule reports whether the remote value at path matches any of
// the resource's configured backend-default rules (and the rule's allowed values,
// if specified).
func (cfg *ResourceLifecycleConfig) matchesBackendDefaultRule(path *structpath.PathNode, remote any) bool {
	for _, rule := range cfg.BackendDefaults {
		if !path.HasPatternPrefix(rule.Field) {
			continue
		}
		if len(rule.Values) == 0 {
			return true
		}
		if MatchesAllowedValue(remote, rule.Values) {
			return true
		}
	}
	return false
}

// MatchesAllowedValue checks if the remote value matches one of the allowed JSON values.
// Each json.RawMessage is unmarshaled into the same type as remote for comparison.
func MatchesAllowedValue(remote any, values []json.RawMessage) bool {
	remoteType := reflect.TypeOf(remote)
	for _, raw := range values {
		candidate := reflect.New(remoteType).Interface()
		if err := json.Unmarshal(raw, candidate); err != nil {
			continue
		}
		if structdiff.IsEqual(remote, reflect.ValueOf(candidate).Elem().Interface()) {
			return true
		}
	}
	return false
}
