package dresources

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/databricks/cli/libs/structs/structpath"
	"go.yaml.in/yaml/v3"
)

// FieldRule represents a field path with its reason for inclusion.
type FieldRule struct {
	Field  *structpath.PatternNode `yaml:"field"`
	Reason string                  `yaml:"reason"`
}

// BackendDefaultRule represents a field that may be set by the backend as a default.
// When old and new are nil but remote is set, and the field matches, the change is skipped.
// If Values is non-empty, the remote value must match one of the allowed values.
type BackendDefaultRule struct {
	Field  *structpath.PatternNode `yaml:"field"`
	Values []json.RawMessage      `yaml:"values,omitempty"`
}

// UnmarshalYAML implements custom YAML unmarshaling for BackendDefaultRule.
// Values are parsed from native YAML types and stored as JSON bytes.
func (b *BackendDefaultRule) UnmarshalYAML(unmarshal func(any) error) error {
	type helper struct {
		Field  *structpath.PatternNode `yaml:"field"`
		Values []any                   `yaml:"values,omitempty"`
	}
	var h helper
	if err := unmarshal(&h); err != nil {
		return err
	}
	b.Field = h.Field
	for _, v := range h.Values {
		raw, err := json.Marshal(v)
		if err != nil {
			return err
		}
		b.Values = append(b.Values, json.RawMessage(raw))
	}
	return nil
}

// ResourceLifecycleConfig defines lifecycle behavior for a resource type.
type ResourceLifecycleConfig struct {
	// IgnoreRemoteChanges: field patterns where remote changes are ignored (output-only, policy-set).
	IgnoreRemoteChanges []FieldRule `yaml:"ignore_remote_changes,omitempty"`

	// IgnoreLocalChanges: field patterns where local changes are ignored (can't be updated via API).
	IgnoreLocalChanges []FieldRule `yaml:"ignore_local_changes,omitempty"`

	// RecreateOnChanges: field patterns that trigger delete + create when changed.
	RecreateOnChanges []FieldRule `yaml:"recreate_on_changes,omitempty"`

	// UpdateIDOnChanges: field patterns that trigger UpdateWithID when changed.
	UpdateIDOnChanges []FieldRule `yaml:"update_id_on_changes,omitempty"`

	// BackendDefaults: fields where the backend may set defaults.
	// When old and new are nil but remote is set, and the remote value matches allowed values (if specified), the change is skipped.
	BackendDefaults []BackendDefaultRule `yaml:"backend_defaults,omitempty"`
}

// Config is the root configuration structure for resource lifecycle behavior.
type Config struct {
	Resources map[string]ResourceLifecycleConfig `yaml:"resources"`
}

//go:embed resources.yml
var resourcesYAML []byte

//go:embed resources.generated.yml
var resourcesGeneratedYAML []byte

var (
	configOnce          sync.Once
	globalConfig        *Config
	generatedConfigOnce sync.Once
	generatedConfig     *Config
	empty               = ResourceLifecycleConfig{
		IgnoreRemoteChanges: nil,
		IgnoreLocalChanges:  nil,
		RecreateOnChanges:   nil,
		UpdateIDOnChanges:   nil,
	}
)

// MustLoadConfig loads and parses the embedded resources.yml configuration.
// The config is loaded once and cached for subsequent calls.
// Panics if the embedded YAML is invalid.
func MustLoadConfig() *Config {
	configOnce.Do(func() {
		globalConfig = &Config{
			Resources: nil,
		}
		if err := yaml.Unmarshal(resourcesYAML, globalConfig); err != nil {
			panic(err)
		}
	})
	return globalConfig
}

// MustLoadGeneratedConfig loads and parses the embedded resources.generated.yml configuration.
// The config is loaded once and cached for subsequent calls.
// Panics if the embedded YAML is invalid.
func MustLoadGeneratedConfig() *Config {
	generatedConfigOnce.Do(func() {
		generatedConfig = &Config{
			Resources: nil,
		}
		if err := yaml.Unmarshal(resourcesGeneratedYAML, generatedConfig); err != nil {
			panic(err)
		}
	})
	return generatedConfig
}

// GetResourceConfig returns the lifecycle config for a given resource type.
// Returns nil if the resource type has no configuration.
func GetResourceConfig(resourceType string) *ResourceLifecycleConfig {
	cfg := MustLoadConfig()
	if rc, ok := cfg.Resources[resourceType]; ok {
		return &rc
	}
	return &empty
}

// GetGeneratedResourceConfig returns the generated lifecycle config for a given resource type.
// Returns nil if the resource type has no configuration.
func GetGeneratedResourceConfig(resourceType string) *ResourceLifecycleConfig {
	cfg := MustLoadGeneratedConfig()
	if rc, ok := cfg.Resources[resourceType]; ok {
		return &rc
	}
	return &empty
}
