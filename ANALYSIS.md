# Grant Types Simplification Analysis

## Current State

There are **5 custom Grant types** in `bundle/config/resources/`:

| Type | Used by | Privilege type |
|------|---------|----------------|
| `Grant` (grant.go) | `RegisteredModel` | `[]string` |
| `SchemaGrant` | `Schema` | `[]SchemaGrantPrivilege` |
| `CatalogGrant` | `Catalog` | `[]CatalogGrantPrivilege` |
| `ExternalLocationGrant` | `ExternalLocation` | `[]ExternalLocationGrantPrivilege` |
| `VolumeGrant` | `Volume` | `[]VolumeGrantPrivilege` |

Each resource-specific Grant type is paired with a `*GrantPrivilege` enum + `Values()` method
(e.g., `SchemaGrantPrivilege` with 16 constants). These are ~150 lines of boilerplate per resource type.

There is also `GrantAssignment` in `bundle/direct/dresources/grants.go` which is essentially
`{ Principal string, Privileges []catalog.Privilege }`, plus reflection-based conversion from
the various Grant types to `GrantAssignment`.

## SDK Type Available

`catalog.PrivilegeAssignment` from the SDK has:

```go
type PrivilegeAssignment struct {
    Principal       string           `json:"principal,omitempty"`
    Privileges      []Privilege      `json:"privileges,omitempty"`
    ForceSendFields []string         `json:"-"`
}
```

## Feasibility Analysis

Replacing all custom Grant types with `catalog.PrivilegeAssignment` is possible, with tradeoffs:

1. **Loss of per-resource privilege enums**: The resource-specific types (e.g., `SchemaGrantPrivilege`)
   restrict which privileges appear in JSON schema documentation/validation. Using `catalog.Privilege`
   gives the full superset. Users could put invalid privileges and only get errors at API call time.

2. **JSON tag difference**: Custom types use `json:"privileges"` (no omitempty); SDK uses
   `json:"privileges,omitempty"`. This could affect bundle YAML parsing behavior.

3. **`Grant` is already inconsistent**: `RegisteredModel.Grants` uses `[]Grant` with
   `Privileges []string` instead of typed privileges — it's already the odd one out.

4. **`dresources.GrantAssignment`** could also be replaced with `catalog.PrivilegeAssignment`
   directly, eliminating the reflection-based conversion in `PrepareGrantsInputConfig`.

5. **Schema generation**: The tests in `schema_test.go` validate that the privilege enums match
   SDK values — this test infrastructure would be simplified away.

## Plan

Replace all custom Grant types (`Grant`, `SchemaGrant`, `CatalogGrant`, `ExternalLocationGrant`,
`VolumeGrant`) with `catalog.PrivilegeAssignment`. Also replace `dresources.GrantAssignment`
with `catalog.PrivilegeAssignment` and remove the reflection-based conversion.

Files affected:
- `bundle/config/resources/grant.go` — remove entirely
- `bundle/config/resources/schema.go` — remove `SchemaGrant*` types, use `catalog.PrivilegeAssignment`
- `bundle/config/resources/catalog.go` — remove `CatalogGrant*` types, use `catalog.PrivilegeAssignment`
- `bundle/config/resources/external_location.go` — remove `ExternalLocationGrant*` types
- `bundle/config/resources/volume.go` — remove `VolumeGrant*` types
- `bundle/config/resources/registered_model.go` — update `Grants` field type
- `bundle/direct/dresources/grants.go` — replace `GrantAssignment`, remove reflection conversion
- `bundle/config/resources/schema_test.go` — update/remove privilege exhaustiveness tests
- `bundle/config/resources/volume_test.go` — update tests
- Various test files referencing the old types
