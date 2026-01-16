package dresources

import (
	_ "embed"
	"sync"

	"github.com/databricks/cli/libs/structs/structpath"
	"gopkg.in/yaml.v3"
)

// ResourceLifecycleConfig defines lifecycle behavior for a resource type.
type ResourceLifecycleConfig struct {
	// IgnoreRemoteChanges: field patterns where remote changes are ignored (output-only, policy-set).
	IgnoreRemoteChanges []*structpath.PathNode `yaml:"ignore_remote_changes,omitempty"`

	// IgnoreLocalChanges: if true, local config changes will be ignored (read-only resource).
	IgnoreLocalChanges bool `yaml:"ignore_local_changes,omitempty"`

	// RecreateOnChanges: field patterns that trigger delete + create when changed.
	RecreateOnChanges []*structpath.PathNode `yaml:"recreate_on_changes,omitempty"`

	// UpdateIDOnChanges: field patterns that trigger UpdateWithID when changed.
	UpdateIDOnChanges []*structpath.PathNode `yaml:"update_id_on_changes,omitempty"`
}

// Config is the root configuration structure for resource lifecycle behavior.
type Config struct {
	Resources map[string]ResourceLifecycleConfig `yaml:"resources"`
}

//go:embed resources.yml
var resourcesYAML []byte

var (
	configOnce   sync.Once
	globalConfig *Config
	configErr    error
)

// LoadConfig loads and parses the embedded resources.yml configuration.
// The config is loaded once and cached for subsequent calls.
func LoadConfig() (*Config, error) {
	configOnce.Do(func() {
		globalConfig = &Config{
			Resources: nil,
		}
		configErr = yaml.Unmarshal(resourcesYAML, globalConfig)
	})
	return globalConfig, configErr
}

// GetResourceConfig returns the lifecycle config for a given resource type.
// Returns nil if the resource type has no configuration.
func GetResourceConfig(resourceType string) *ResourceLifecycleConfig {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil {
		return nil
	}
	if rc, ok := cfg.Resources[resourceType]; ok {
		return &rc
	}
	return nil
}
