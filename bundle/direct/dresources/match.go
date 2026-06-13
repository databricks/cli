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
