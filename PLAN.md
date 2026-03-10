# Implementation Plan: `__EMBED__` Convention for Struct Walkers

## Problem

In bundle config, permissions are a direct slice on the resource:
```
resources.jobs.foo.permissions[0].user_name
```

In direct engine state, permissions are wrapped in `PermissionsState`:
```go
type PermissionsState struct {
    ObjectID    string                     `json:"object_id"`
    Permissions []iam.AccessControlRequest `json:"permissions,omitempty"`
}
```

This creates an extra level in the path:
```
resources.jobs.foo.permissions.permissions[0].user_name
```

When reference resolution strips the prefix `resources.jobs.foo.permissions` and tries to navigate `[0].service_principal_name` within PermissionsState, it fails with "cannot index struct" because the struct expects `permissions[0].service_principal_name`.

## Solution

Introduce `__EMBED__` json tag convention: when a struct field has `json:"__EMBED__"`, all struct walkers in `libs/structs/` treat it as transparent - they don't add the field name to the path. The field's contents appear directly at the parent level.

With `__EMBED__`, PermissionsState becomes:
```go
type PermissionsState struct {
    ObjectID    string                     `json:"object_id"`
    Permissions []iam.AccessControlRequest `json:"__EMBED__,omitempty"`
}
```

Now `[0].user_name` navigates correctly through the embedded slice.

## Implementation Steps

### 1. `libs/structs/structtag/jsontag.go`
- Add `const EmbedTagName = "__EMBED__"`.
- Add `IsEmbed()` method on `JSONTag`.
- Add test.

### 2. `libs/structs/structwalk/walk.go` — `walkStruct`
- After parsing json tag, check if `jsonTag.IsEmbed()`.
- If so, walk the field value at the parent path (don't add field name to path), like anonymous embedding.
- Still respect omitempty.
- Add test case to existing table tests.

### 3. `libs/structs/structwalk/walktype.go` — `walkTypeStruct`
- Same logic: if json tag name is `__EMBED__`, walk at parent path level.
- Add test case to existing table tests.

### 4. `libs/structs/structaccess/get.go`
- In `getValue`: when `cur` is a struct and the path node is an index or key-value selector, find the `__EMBED__` field (a slice) and navigate into it.
- In `findFieldInStruct`: skip fields with `__EMBED__` tag name (they're not accessible by name).
- Add helper `findEmbedField(v reflect.Value) reflect.Value`.
- Add test cases to existing tests.

### 5. `libs/structs/structaccess/set.go`
- In `setValueAtNode`: when parent is a struct and the node is an index, resolve through `__EMBED__` field.
- Add test cases.

### 6. `libs/structs/structaccess/typecheck.go`
- In `validateNodeSlice`: when type is struct and path expects index/bracket-star/key-value, find `__EMBED__` field type.
- In `FindStructFieldByKeyType`: skip `__EMBED__` fields from string key lookups.
- Add helper `findEmbedFieldType(t reflect.Type) reflect.Type`.
- Add test cases.

### 7. `libs/structs/structdiff/diff.go`
- In `diffStruct`: when field has `__EMBED__` tag, use parent path (don't add field name).
- Still handle omitempty/zero/forced as normal.
- Add test cases.

### 8. `bundle/direct/dresources/permissions.go`
- Change `Permissions` tag to `json:"__EMBED__,omitempty"`.
- Update `KeyedSlices()` to use `""` as key instead of `"permissions"` (since the slice is now at root level).

### 9. Acceptance Tests
- Enable direct engine in `acceptance/bundle/apps/job_permissions/test.toml`.
- Add new acceptance tests for:
  - Referencing a permission field from another resource.
  - Referencing another resource from a permission field.
  - Permission indices from remote backend (index not in local config).

### 10. Validation
- `make generate` runs clean.
- `make build && make test-update` succeeds.
- `make test-update-aws` for integration tests.
