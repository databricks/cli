## Summary

Struct walkers in `libs/structs/` now detect embedded slices by the Go field name `EmbeddedSlice` instead of the `__EMBED__` json tag convention. This simplifies PermissionsState by removing custom JSON marshaling and the dual-type workaround.

Key changes:
- Replace `__EMBED__` json tag with `EmbeddedSlice` field name as the embed signal
- Remove `permissionsStateJSON`, `MarshalJSON()`, `UnmarshalJSON()` from PermissionsState
- Fix refschema output: `permissions.[*]` → `permissions[*]` (no dot before brackets)
- Fix refschema output: `permissions.permissions[*]` → `permissions[*]` (no duplicate prefix)
- Enable direct engine for `job_permissions` acceptance test (previously broken)

## Test plan
- [x] All 4579 unit tests pass
- [x] Acceptance tests pass for both terraform and direct engines
- [x] `job_permissions` test now passes on direct engine
- [x] `make checks fmt lint` clean
