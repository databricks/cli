# Plan: Replace FieldTriggers with Declarative YAML Config

## Overview

Replace the programmatic `FieldTriggers()` method on each resource with a declarative YAML configuration file (`dresources/resources.yml`) that is embedded into the binary.

## Current State

Currently, each resource optionally implements:
```go
func (*ResourceVolume) FieldTriggers() map[string]deployplan.ActionType {
    return map[string]deployplan.ActionType{
        "catalog_name":     deployplan.Recreate,
        "name":             deployplan.UpdateWithID,
    }
}
```

Resources with FieldTriggers:
- `volume.go`: catalog_name, schema_name, storage_location, volume_type → Recreate; name → UpdateWithID
- `schema.go`: name, catalog_name, storage_root → Recreate
- `pipeline.go`: storage, ingestion_definition.connection_name, ingestion_definition.ingestion_gateway_id → Recreate
- `model_serving_endpoint.go`: name, description, route_optimized, auto_capture configs → Recreate
- `secret_scope_acls.go`: scope_name → UpdateWithID
- `experiment.go`: name, artifact_location → Recreate
- `model.go`: name → Recreate
- `quality_monitor.go`: output_schema_name, assets_dir → Recreate
- `registered_model.go`: catalog_name, schema_name, name → Recreate
- `app.go`: name → Recreate
- `secret_scope.go`: name, backend_type → Recreate

## Proposed YAML Schema

```yaml
# dresources/resources.yml
resources:
  jobs:
    # Example: ignore server-set fields
    ignore_remote_changes:
      - effective_budget_policy_id
      - "tags[key='server_tag']"

  volumes:
    recreate_on_changes:
      - catalog_name
      - schema_name
      - storage_location
      - volume_type
    update_id_on_changes:
      - name

  pipelines:
    recreate_on_changes:
      - storage
      - ingestion_definition.connection_name
      - ingestion_definition.ingestion_gateway_id

  schemas:
    recreate_on_changes:
      - name
      - catalog_name
      - storage_root

  # Permissions and grants are top-level entries
  permissions:
    # config for all permissions resources

  grants:
    # config for all grants resources
```

## Pattern Syntax

Patterns use `libs/structs/structpath` syntax:

- **Dot notation**: `foo.bar` - nested fields
- **Bracket notation**: `['field-name']` - fields with special characters
- **Array indices**: `[0]`, `[1]` - specific array elements
- **Wildcards**: `.*` (any field) and `[*]` (any array element)
- **Key-value matching**: `[key='value']` - match array/map elements by key

Examples:
- `name` - simple field
- `config.auto_capture_config.catalog_name` - nested field
- `tags[key='server_tag']` - specific tag by key
- `tasks[*].task_key` - all task keys in tasks array

## Implementation Steps

### Step 1: Define Go Types for the Config

Create `bundle/direct/dresources/config.go`:

```go
package dresources

import "github.com/databricks/cli/libs/structs/structpath"

// ResourceLifecycleConfig defines lifecycle behavior for a resource type.
type ResourceLifecycleConfig struct {
    // IgnoreRemoteChanges: fields where remote changes are ignored (output-only, policy-set)
    IgnoreRemoteChanges []*structpath.PathNode `yaml:"ignore_remote_changes,omitempty"`

    // IgnoreLocalChanges: local config changes will be ignored (read-only resource)
    IgnoreLocalChanges bool `yaml:"ignore_local_changes,omitempty"`

    // RecreateOnChanges: field changes trigger delete + create
    RecreateOnChanges []*structpath.PathNode `yaml:"recreate_on_changes,omitempty"`

    // UpdateIDOnChanges: field changes trigger UpdateWithID
    UpdateIDOnChanges []*structpath.PathNode `yaml:"update_id_on_changes,omitempty"`
}

// Config is the root configuration structure.
type Config struct {
    Resources map[string]ResourceLifecycleConfig `yaml:"resources"`
}
```

Note: `structpath.PathNode` implements `yaml.Marshaler` and `yaml.Unmarshaler` interfaces,
enabling automatic parsing during config load with early validation of pattern syntax.

### Step 2: Create the YAML Config File

Create `bundle/direct/dresources/resources.yml` with all current FieldTriggers data migrated to the new format.

### Step 3: Embed and Load the Config

Create `bundle/direct/dresources/config_loader.go`:

```go
package dresources

import (
    _ "embed"
    "sync"
    "gopkg.in/yaml.v3"
)

//go:embed resources.yml
var resourcesYAML []byte

var (
    configOnce sync.Once
    globalConfig *Config
    configErr error
)

func LoadConfig() (*Config, error) {
    configOnce.Do(func() {
        globalConfig = &Config{}
        configErr = yaml.Unmarshal(resourcesYAML, globalConfig)
    })
    return globalConfig, configErr
}

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
```

### Step 4: Modify Adapter to Use Config Instead of FieldTriggers

Update `adapter.go`:

1. Add a field to store the resource config:
   ```go
   type Adapter struct {
       // ... existing fields ...
       resourceConfig *ResourceLifecycleConfig
   }
   ```

2. In `NewAdapter`, load from config instead of calling FieldTriggers():
   ```go
   // Replace FieldTriggers loading with:
   adapter.resourceConfig = GetResourceConfig(resourceType)
   adapter.fieldTriggers = adapter.buildFieldTriggersFromConfig()
   ```

3. Add method to convert config to fieldTriggers map:
   ```go
   func (a *Adapter) buildFieldTriggersFromConfig() map[string]deployplan.ActionType {
       if a.resourceConfig == nil {
           return map[string]deployplan.ActionType{}
       }
       result := make(map[string]deployplan.ActionType)
       for _, field := range a.resourceConfig.RecreateOnChanges {
           result[field] = deployplan.Recreate
       }
       for _, field := range a.resourceConfig.UpdateIDOnChanges {
           result[field] = deployplan.UpdateWithID
       }
       return result
   }
   ```

### Step 5: Add Pattern Matching for `ignore_remote_changes`

Add pattern matching function to `libs/structs/structpath/path.go`:

```go
// MatchPattern checks if a path matches a pattern with wildcards.
// Pattern can contain:
//   - .* to match any single field
//   - [*] to match any array index
//   - [key='value'] to match specific key-value pairs
func MatchPattern(path, pattern string) bool {
    // Implementation using structpath.Parse for both path and pattern,
    // then comparing node by node with wildcard support
}
```

### Step 6: Update `addPerFieldActions` for New Config Options

Modify `bundle_plan.go:addPerFieldActions` to handle:
- `ignore_remote_changes`: Skip action if field matches any pattern (using MatchPattern)
- `ignore_local_changes`: Skip all local changes for the resource

### Step 7: Remove FieldTriggers Methods from Resources

Remove `FieldTriggers()` method from all resource files:
- volume.go
- schema.go
- pipeline.go
- model_serving_endpoint.go
- secret_scope_acls.go
- experiment.go
- model.go
- quality_monitor.go
- registered_model.go
- app.go
- secret_scope.go

### Step 8: Update IResource Interface

Remove FieldTriggers from the IResource interface in `adapter.go`.

### Step 9: Update Tests

1. Update `TestFieldTriggers` in `all_test.go` to validate config loading
2. Update `TestFieldTriggersNoUpdateWhenNotImplemented` to work with new config
3. Add tests for new config features (ignore_remote_changes, etc.)
4. Add tests for pattern matching in structpath

### Step 10: Update Validation

The adapter validation currently checks:
- `DoUpdateWithID` is implemented if FieldTriggers has `update_with_id`
- `DoUpdateWithID` has trigger if implemented

Update this to validate against the YAML config instead.

## File Changes Summary

| File | Action |
|------|--------|
| `libs/structs/structpath/path.go` | Modify - Add MatchPattern function |
| `libs/structs/structpath/path_test.go` | Modify - Add MatchPattern tests |
| `dresources/config.go` | New - Go types for config |
| `dresources/resources.yml` | New - YAML config file |
| `dresources/adapter.go` | Modify - Remove FieldTriggers loading, add config-based approach |
| `dresources/volume.go` | Modify - Remove FieldTriggers() |
| `dresources/schema.go` | Modify - Remove FieldTriggers() |
| `dresources/pipeline.go` | Modify - Remove FieldTriggers() |
| `dresources/model_serving_endpoint.go` | Modify - Remove FieldTriggers() |
| `dresources/secret_scope_acls.go` | Modify - Remove FieldTriggers() |
| `dresources/experiment.go` | Modify - Remove FieldTriggers() |
| `dresources/model.go` | Modify - Remove FieldTriggers() |
| `dresources/quality_monitor.go` | Modify - Remove FieldTriggers() |
| `dresources/registered_model.go` | Modify - Remove FieldTriggers() |
| `dresources/app.go` | Modify - Remove FieldTriggers() |
| `dresources/secret_scope.go` | Modify - Remove FieldTriggers() |
| `dresources/all_test.go` | Modify - Update tests |
| `direct/bundle_plan.go` | Modify - Handle new config options |

## Migration Strategy

1. First implement config loading alongside existing FieldTriggers
2. Add new config features (ignore_remote_changes, etc.)
3. Migrate all existing FieldTriggers data to YAML
4. Remove FieldTriggers methods and update interface
5. Update tests

## Design Decisions

1. **Permissions and grants**: Top-level entries `permissions` and `grants` in the YAML config (not nested under parent resources)
2. **Pattern syntax**: Uses `libs/structs/structpath` syntax with wildcards (`.*`, `[*]`) and key-value matching (`[key='value']`)
3. **PathNode in config**: Config uses `[]*structpath.PathNode` directly instead of `[]string` for early validation and parse-once semantics
