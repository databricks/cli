package configsync

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"slices"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

type (
	skipAlways           struct{}
	skipIfZeroOrNil      struct{}
	skipIfEmptyOrDefault struct {
		defaults map[string]any
	}
)

// One file per resource type under rules/, named after the type's key under
// resources:, holds the sync rules of that type. Fields in a file are relative
// to the resource. A resource type without rules has no file.
//
//go:embed rules/*.yml
var rulesFS embed.FS

// resourceRulesFile is the content of one file under rules/.
type resourceRulesFile struct {
	// ServerSideDefaults are the hardcoded server-side defaults.
	// This is a temporary solution until the bundle plan issue is resolved.
	ServerSideDefaults []serverSideDefaultRule `yaml:"server_side_defaults"`

	// ResetValues are the values that should be used to reset CLI-defaulted fields.
	// If CLI-defaulted field is changed on remote and should be disabled (e.g. queueing
	// disabled -> remote field is nil) we can't define it in the config as "null"
	// because CLI default will be applied again.
	ResetValues []fieldValueRule `yaml:"reset_values"`

	// PrefixedNameFields are the name fields where the name prefix (e.g. "[dev user] ")
	// is applied during deployment and should be stripped when syncing remote changes
	// back to config.
	PrefixedNameFields []string `yaml:"prefixed_name_fields"`
}

// serverSideDefaultRule marks a remote field value as a default. With skip "always"
// the field is always considered a default regardless of value; with "zero_or_nil"
// when it is nil or 0; with "empty_or_default" when it is an empty map or equals the
// map in value. Without skip, it is compared with value using reflect.DeepEqual.
type serverSideDefaultRule struct {
	Field string `yaml:"field"`
	Skip  string `yaml:"skip"`
	Value any    `yaml:"value"`
}

type fieldValueRule struct {
	Field string `yaml:"field"`
	Value any    `yaml:"value"`
}

// fieldPattern pairs a full field pattern, such as "resources.jobs.*.queue", with a value.
type fieldPattern struct {
	pattern string
	value   any
}

type syncRules struct {
	serverSideDefaults []fieldPattern
	resetValues        []fieldPattern
	prefixedNameFields []string
}

// loadSyncRules parses the files under rules/ in order of resource key, keeping the
// order of the rules within each file.
var loadSyncRules = sync.OnceValue(func() syncRules {
	var rules syncRules
	for _, name := range rulesFileNames() {
		data, err := rulesFS.ReadFile(name)
		if err != nil {
			panic(err)
		}
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoder.KnownFields(true)
		var file resourceRulesFile
		if err := decoder.Decode(&file); err != nil {
			panic(fmt.Errorf("%s: %w", name, err))
		}

		prefix := "resources." + rulesResourceType(name) + ".*."
		for _, rule := range file.ServerSideDefaults {
			expected, err := rule.expected()
			if err != nil {
				panic(fmt.Errorf("%s: %s: %w", name, rule.Field, err))
			}
			rules.serverSideDefaults = append(rules.serverSideDefaults, fieldPattern{pattern: prefix + rule.Field, value: expected})
		}
		for _, rule := range file.ResetValues {
			if rule.Value == nil {
				panic(fmt.Errorf("%s: %s: value is required", name, rule.Field))
			}
			rules.resetValues = append(rules.resetValues, fieldPattern{pattern: prefix + rule.Field, value: asRemoteValue(rule.Value)})
		}
		for _, field := range file.PrefixedNameFields {
			rules.prefixedNameFields = append(rules.prefixedNameFields, prefix+field)
		}
	}
	return rules
})

// rulesFileNames returns the files under rules/, sorted.
func rulesFileNames() []string {
	names, err := fs.Glob(rulesFS, "rules/*.yml")
	if err != nil {
		panic(err)
	}
	slices.Sort(names)
	return names
}

// rulesResourceType returns the resource type a file under rules/ holds the rules of.
func rulesResourceType(name string) string {
	return strings.TrimSuffix(strings.TrimPrefix(name, "rules/"), ".yml")
}

// expected returns what shouldSkipField compares a remote value with.
func (r serverSideDefaultRule) expected() (any, error) {
	var expected any
	switch r.Skip {
	case "always":
		expected = skipAlways{}
	case "zero_or_nil":
		expected = skipIfZeroOrNil{}
	case "empty_or_default":
		defaults, ok := asRemoteValue(r.Value).(map[string]any)
		if !ok {
			return nil, fmt.Errorf("skip %q requires a map value", r.Skip)
		}
		return skipIfEmptyOrDefault{defaults: defaults}, nil
	case "":
		if r.Value == nil {
			return nil, errors.New("value is required without skip")
		}
		return asRemoteValue(r.Value), nil
	default:
		return nil, fmt.Errorf("unknown skip %q", r.Skip)
	}
	if r.Value != nil {
		return nil, fmt.Errorf("skip %q takes no value", r.Skip)
	}
	return expected, nil
}

// asRemoteValue converts a value decoded from YAML to the types normalizeValue gives
// remote values, so that reflect.DeepEqual can compare them: YAML integers decode as
// int, remote integers normalize to int64.
func asRemoteValue(v any) any {
	switch v := v.(type) {
	case int:
		return int64(v)
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, value := range v {
			out[key] = asRemoteValue(value)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, value := range v {
			out[i] = asRemoteValue(value)
		}
		return out
	}
	return v
}

// shouldSkipField checks if a field should be skipped in change detection.
// When hasConfigValue is true (field is set in config or saved state), only
// "always skip" fields are skipped. Backend defaults are only skipped when the
// field is not in config/state, matching the behavior of shouldSkipBackendDefault
// in the direct deployment engine.
func shouldSkipField(path string, value any, hasConfigValue bool) bool {
	for _, rule := range loadSyncRules().serverSideDefaults {
		expected := rule.value
		if matchPattern(rule.pattern, path) {
			if _, ok := expected.(skipAlways); ok {
				return true
			}
			if hasConfigValue {
				return false
			}
			if _, ok := expected.(skipIfZeroOrNil); ok {
				return value == nil || value == int64(0)
			}
			if marker, ok := expected.(skipIfEmptyOrDefault); ok {
				m, ok := value.(map[string]any)
				if !ok {
					return false
				}
				if len(m) == 0 {
					return true
				}
				return reflect.DeepEqual(m, marker.defaults)
			}
			return reflect.DeepEqual(value, expected)
		}
	}
	return false
}

func matchPattern(pattern, path string) bool {
	patternParts := strings.Split(pattern, ".")
	pathParts := strings.Split(path, ".")
	return matchParts(patternParts, pathParts)
}

func matchParts(patternParts, pathParts []string) bool {
	if len(patternParts) == 0 && len(pathParts) == 0 {
		return true
	}
	if len(patternParts) == 0 || len(pathParts) == 0 {
		return false
	}

	patternPart := patternParts[0]
	pathPart := pathParts[0]

	if patternPart == "*" {
		return matchParts(patternParts[1:], pathParts[1:])
	}

	if strings.Contains(patternPart, "[*]") {
		prefix, _, _ := strings.Cut(patternPart, "[*]")

		if strings.HasPrefix(pathPart, prefix) && strings.Contains(pathPart, "[") {
			return matchParts(patternParts[1:], pathParts[1:])
		}
		return false
	}

	if patternPart == pathPart {
		return matchParts(patternParts[1:], pathParts[1:])
	}

	return false
}

// resetValueIfNeeded returns the reset value of the CLI-defaulted field at path, or
// value for any other field.
func resetValueIfNeeded(path string, value any) any {
	for _, rule := range loadSyncRules().resetValues {
		if matchPattern(rule.pattern, path) {
			return rule.value
		}
	}
	return value
}

// stripNamePrefix strips the configured name prefix from name field values
// so that the raw (unprefixed) name is written back to the config YAML.
func stripNamePrefix(path string, value any, prefix string) any {
	if prefix == "" {
		return value
	}

	s, ok := value.(string)
	if !ok {
		return value
	}

	for _, pattern := range loadSyncRules().prefixedNameFields {
		if matchPattern(pattern, path) {
			return strings.TrimPrefix(s, prefix)
		}
	}

	return value
}
