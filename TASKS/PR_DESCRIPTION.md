## Changes
- Add `EmbeddedSlice` field name convention to struct walkers in `libs/structs/` — when a struct field is named `EmbeddedSlice`, walkers treat it as transparent (no path segment added), so its elements appear directly at the parent path
- Apply this to `PermissionsState`: rename `Permissions` field to `EmbeddedSlice`, making state paths like `resources.jobs.foo.permissions[0]` match input config paths (previously `resources.jobs.foo.permissions.permissions[0]`)
- Replace `[]iam.AccessControlRequest` with `[]StatePermission` in `PermissionsState` — uses bundle config field name `level` instead of SDK's `permission_level`, with backward-compatible migration for old state files
- Fix refschema output: `permissions.[*]` → `permissions[*]` (remove spurious dot before bracket)
- Enable `job_permissions` acceptance test for direct engine
- Fix reference resolution for permission sub-resources: `${resources.jobs.foo.permissions[0].level}` now correctly routes to the permissions sub-resource node

## Why

The direct deployment engine's permissions state used a wrapper struct that added an extra `permissions` segment to paths. This caused a mismatch with input config paths, preventing dependency tracking between permissions and their parent resources.

Additionally, cross-resource permission references (e.g., `${resources.jobs.job_b.permissions[0].level}`) did not work because the reference resolver assumed all resources use 3-component keys (`resources.type.name`), but permissions are sub-resources with 4-component keys (`resources.type.name.permissions`).

The state also used `iam.AccessControlRequest` with `permission_level` field, while the bundle config uses `level`. This required field remapping. By switching to a state-specific `StatePermission` type that matches the config field name, remapping is eliminated. Old state files with `permission_level` are automatically migrated during deserialization.

## Tests
- Extended existing table-driven unit tests in `structwalk/`, `structaccess/`, and `structdiff/` with EmbeddedSlice cases
- Updated `acceptance/bundle/apps/job_permissions` to run on both terraform and direct engines
- Updated `acceptance/bundle/refschema` output to reflect corrected paths
- Updated direct engine plan output files for permissions path changes
- Added `acceptance/bundle/resource_deps/permission_ref/` test demonstrating cross-resource permission level references between jobs
- Added `acceptance/bundle/state/permission_level_migration/` test verifying backward-compatible migration of old `permission_level` state to `level`
