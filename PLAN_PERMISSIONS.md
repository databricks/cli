# Plan: Simplify Permissions (analogous to simpler-grant)

## What Was Done in simpler-grant

### Problem
Each resource type had its own custom Grant type with a resource-specific privilege enum:
- `Grant` (generic, `Privileges []string`) for `registered_model`
- `SchemaGrant` with `SchemaGrantPrivilege` enum for `schema`
- `CatalogGrant` with `CatalogGrantPrivilege` enum for `catalog`
- `ExternalLocationGrant` with `ExternalLocationGrantPrivilege` enum for `external_location`
- `VolumeGrant` with `VolumeGrantPrivilege` enum for `volume`

The `PrepareGrantsInputConfig` used reflection to convert any of these custom types into `[]catalog.PrivilegeAssignment` for the direct engine.

The SDK already had `catalog.PrivilegeAssignment` (principal + `[]catalog.Privilege`) which covers all UC privileges.

### What Was Changed
1. **Go**: Deleted `grant.go` and removed 5 custom Grant types + their privilege enums from `catalog.go`, `schema.go`, `volume.go`, `external_location.go`, `registered_model.go`. Replaced all `Grants` fields with `[]catalog.PrivilegeAssignment`.
2. **Go**: Replaced the reflection-based `PrepareGrantsInputConfig` with a simple type assertion `inputConfig.(*[]catalog.PrivilegeAssignment)`.
3. **Go**: Deleted `TestSchemaGrantPrivilegesExhaustive` test (which kept the old enum in sync with the SDK).
4. **Schema**: Updated `annotations.yml` (removed custom type annotations), `annotations_openapi_overrides.yml` (removed overrides for custom types), regenerated `jsonschema.json`.
5. **Python**: Removed per-resource grant/privilege model files (`schema_grant.py`, `schema_grant_privilege.py`, `catalog_grant.py`, etc.), added shared `privilege.py` and `privilege_assignment.py` under each resource package.
6. **Test data bug**: `testdata/pass/ml.yml` used `INSERT` as a privilege, which was accepted with the old `[]string` type but invalid with `catalog.Privilege` enum. Fixed by replacing with `MODIFY`.

### Net Result
- Deleted ~950 lines, added ~111 lines
- No behavioral change — the SDK's `catalog.Privilege` enum is a superset of the old per-resource enums (which were manually maintained subsets)

### Issues Encountered
1. **Schema testdata used invalid privilege value** (`INSERT` instead of a valid UC privilege like `MODIFY`). The old `Grant` type used `[]string` with no enum validation, hiding this. Only surfaced after switching to the typed SDK enum. Caught by CI (`validate-generated-is-up-to-date`).
   - **Lesson**: After switching from `[]string`/untyped to a typed enum, check all testdata/acceptance files for values that are now invalid.

---

## Plan: Simplify Permissions

### Current State (analogous to the old grants situation)

`bundle/config/resources/permission.go` defines:
- 12 resource-specific permission types: `AlertPermission`, `AppPermission`, `ClusterPermission`, `DashboardPermission`, `DatabaseInstancePermission`, `DatabaseProjectPermission`, `JobPermission`, `MlflowExperimentPermission`, `MlflowModelPermission`, `ModelServingEndpointPermission`, `PipelinePermission`, `SqlWarehousePermission`
- 12 corresponding level types: `AlertPermissionLevel`, `AppPermissionLevel`, etc. — all declared as `string` with no enum values (only `AlertPermissionLevel` has a `Values()` method)
- `IPermission` interface with 4 methods: `GetLevel()`, `GetUserName()`, `GetServicePrincipalName()`, `GetGroupName()`, `GetAPIRequestObjectType()`
- ~180 lines of boilerplate: `GetLevel`/`GetUserName`/`GetServicePrincipalName`/`GetGroupName` implementations repeated 12 times
- `GetAPIRequestObjectType()` per type (maps to API path prefix like `/jobs/`, `/pipelines/`, etc.)

`bundle/direct/dresources/permissions.go` uses reflection to iterate over the slice, casts each element to `IPermission`, extracts level/principal fields, and converts to `[]iam.AccessControlRequest`.

The SDK's `iam.AccessControlRequest` already has: `PermissionLevel iam.PermissionLevel`, `UserName`, `GroupName`, `ServicePrincipalName`.

### Goal

Replace all 12 resource-specific permission types with `iam.AccessControlRequest` directly from the SDK, analogous to how `catalog.PrivilegeAssignment` replaced the custom grant types.

The object type prefix (e.g. `/jobs/`) needs to come from somewhere — currently embedded in each type via `GetAPIRequestObjectType()`. This must be handled differently (see Step 2 below).

### Steps

#### Step 1: Understand how `GetAPIRequestObjectType` is used

`PreparePermissionsInputConfig` uses it to build the `objectIdRef`:
```go
prefix := zeroValueInterface.GetAPIRequestObjectType()
objectIdRef := prefix + "${" + baseNode + ".id}"
```

The mapping is:
| Type | Prefix |
|------|--------|
| AlertPermission | `/alertsv2/` |
| AppPermission | `/apps/` |
| ClusterPermission | `/clusters/` |
| DashboardPermission | `/dashboards/` |
| DatabaseInstancePermission | `/database-instances/` |
| DatabaseProjectPermission | `/database-projects/` |
| JobPermission | `/jobs/` |
| MlflowExperimentPermission | `/experiments/` |
| MlflowModelPermission | `/registered-models/` |
| ModelServingEndpointPermission | `/serving-endpoints/` |
| PipelinePermission | `/pipelines/` |
| SqlWarehousePermission | `/sql/warehouses/` |

This mapping is from resource type name (in the bundle node path) to object type prefix. It can be extracted from the node path (`baseNode`) directly, e.g. `resources.jobs.myjob.permissions` → `/jobs/`.

#### Step 2: Replace IPermission dispatch with a static map

Add a map in `permissions.go` (analogous to `grantResourceToSecurableType` in `grants.go`):
```go
var permissionResourceToObjectType = map[string]string{
    "alerts":                   "/alertsv2/",
    "apps":                     "/apps/",
    "clusters":                 "/clusters/",
    "dashboards":               "/dashboards/",
    "database_instances":       "/database-instances/",
    "postgres_projects":        "/database-projects/",
    "jobs":                     "/jobs/",
    "experiments":              "/experiments/",
    "models":                   "/registered-models/",
    "model_serving_endpoints":  "/serving-endpoints/",
    "pipelines":                "/pipelines/",
    "sql_warehouses":           "/sql/warehouses/",
}
```

Then extract resource type from `baseNode` (e.g. `resources.jobs.myjob` → `jobs`) and look it up.

Note: The existing special cases in `PreparePermissionsInputConfig` for `model_serving_endpoints` (uses `.endpoint_id` not `.id`) and `postgres_projects` (uses `.project_id`) are already handled via string prefix checks and can remain.

#### Step 3: Change resource structs (Go)

In each resource file, change:
```go
Permissions []JobPermission `json:"permissions,omitempty"`
```
to:
```go
Permissions []iam.AccessControlRequest `json:"permissions,omitempty"`
```

Resources affected: `alerts.go`, `apps.go`, `clusters.go`, `dashboard.go`, `database_instance.go`, `job.go`, `mlflow_experiment.go`, `mlflow_model.go`, `model_serving_endpoint.go`, `pipeline.go`, `postgres_project.go`, `sql_warehouses.go`.

Note: `secret_scope.go` uses `SecretScopePermission` (different mechanism — secret ACLs, not the Permissions API). Leave it unchanged.

#### Step 4: Simplify `PreparePermissionsInputConfig` (direct engine)

Replace the reflection-based conversion with a simple type assertion:
```go
permissionsPtr, ok := inputConfig.(*[]iam.AccessControlRequest)
if !ok {
    return nil, fmt.Errorf("expected *[]iam.AccessControlRequest, got %T", inputConfig)
}
permissions := *permissionsPtr
```

Remove the `reflect` import, remove the `IPermission` cast loop.

#### Step 5: Delete `permission.go` content

Remove from `bundle/config/resources/permission.go`:
- All 12 `*PermissionLevel` type declarations
- All 12 `*Permission` struct types
- All `GetAPIRequestObjectType()` methods
- All `GetLevel()`/`GetUserName()`/`GetServicePrincipalName()`/`GetGroupName()` boilerplate
- `IPermission` interface
- The unused `Permission` struct (the generic one at top — check if still used anywhere)

Keep in `permission.go`:
- `PermissionOrder`, `GetLevelScore`, `CompareLevels`, `GetMaxLevel` (used by workspace permissions logic)

#### Step 6: Update schema annotations

In `bundle/internal/schema/annotations.yml`, remove the per-type permission annotations (e.g. `github.com/databricks/cli/bundle/config/resources.AlertPermission:`, etc.) since those types are gone. The `iam.AccessControlRequest` type from the SDK will be used instead, annotated via `annotations_openapi.yml`.

Check `annotations_openapi_overrides.yml` for any overrides on the old permission types — remove those too.

#### Step 7: Regenerate schema and validate

```bash
make schema
```

Verify `bundle/schema/jsonschema.json` and `bundle/schema/jsonschema_for_docs.json` look correct — the `permissions` field on each resource should reference `iam.AccessControlRequest` schema from SDK, which will have `permission_level` (enum of `iam.PermissionLevel` values), `user_name`, `group_name`, `service_principal_name`.

#### Step 8: Update Python models

Remove per-resource permission model files:
- `python/databricks/bundles/jobs/_models/job_permission.py`
- `python/databricks/bundles/jobs/_models/job_permission_level.py`
- `python/databricks/bundles/pipelines/_models/pipeline_permission.py`
- `python/databricks/bundles/pipelines/_models/pipeline_permission_level.py`
- Any other `*_permission.py` / `*_permission_level.py` files

Add shared `access_control_request.py` and `permission_level.py` under affected packages (jobs, pipelines, etc.), generated from SDK types.

Update `__init__.py` files accordingly.

Run: `cd python && make codegen && make test && make lint && make docs`

#### Step 9: Check testdata/acceptance for now-invalid permission level values

**Lesson learned from simpler-grant**: The old `*PermissionLevel` types are declared as `string` with no actual enum validation in the schema (only `AlertPermissionLevel` has `Values()`). After switching to `iam.PermissionLevel`, the schema will enforce the enum. Scan all testdata and acceptance files:

```bash
grep -rn "level:" bundle/internal/schema/testdata/ acceptance/ --include="*.yml"
```

Ensure all `level:` values are valid `iam.PermissionLevel` values:
`CAN_ATTACH_TO`, `CAN_BIND`, `CAN_CREATE`, `CAN_EDIT`, `CAN_EDIT_METADATA`, `CAN_MANAGE`, `CAN_MANAGE_PRODUCTION_VERSIONS`, `CAN_MANAGE_RUN`, `CAN_MANAGE_STAGING_VERSIONS`, `CAN_MONITOR`, `CAN_MONITOR_ONLY`, `CAN_QUERY`, `CAN_READ`, `CAN_RESTART`, `CAN_RUN`, `CAN_USE`, `CAN_VIEW`, `CAN_VIEW_METADATA`, `IS_OWNER`.

Also check Python test files and acceptance scripts.

#### Step 10: Run tests

```bash
make test
make schema   # if schema-related code changed
cd python && make codegen && make test && make lint && make docs
```

---

## Key Differences vs simpler-grant

| Aspect | simpler-grant | simpler-permissions |
|--------|--------------|---------------------|
| Custom types deleted | 5 (one per resource) | ~12 (one per resource type) |
| API object type dispatch | N/A (securable type from node name) | Must map resource type → object type prefix |
| SDK replacement type | `catalog.PrivilegeAssignment` | `iam.AccessControlRequest` |
| Field name change | `privileges []string` → `[]catalog.Privilege` (enum enforced) | `level string` → `iam.PermissionLevel` (enum enforced) |
| Reflection removed | Yes (grants.go) | Yes (permissions.go) |
| Exceptions | None | `secret_scope` (ACL, not Permissions API); `model_serving_endpoints` (uses `.endpoint_id`); `postgres_projects` (uses `.project_id`) |
| Schema enum impact | `INSERT` was invalid (caught by CI) | Any invalid `CAN_*` values will be caught — scan testdata first |

## Risk: `iam.PermissionLevel` is a shared enum

`iam.PermissionLevel` covers ALL resource types' permission levels combined. Some levels are only valid for specific resources (e.g. `CAN_ATTACH_TO` is only for clusters). This is acceptable — same as with `catalog.Privilege` being a superset. The schema will allow any valid `iam.PermissionLevel` on any resource, which is slightly more permissive than the current per-resource enums (but those had no per-resource values anyway — all defined as bare `string`).
