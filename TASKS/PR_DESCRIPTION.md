## Changes
- Add `EmbeddedSlice` field name convention to struct walkers in `libs/structs/` — when a struct field is named `EmbeddedSlice`, walkers treat it as transparent (no path segment added), so its elements appear directly at the parent path
- Apply this to `PermissionsState`: rename `Permissions` field to `EmbeddedSlice`, making state paths like `resources.jobs.foo.permissions[0]` match input config paths (previously `resources.jobs.foo.permissions.permissions[0]`)
- Fix refschema output: `permissions.[*]` → `permissions[*]` (remove spurious dot before bracket)
- Enable `job_permissions` acceptance test for direct engine

## Why

The direct deployment engine's permissions state used a wrapper struct that added an extra `permissions` segment to paths. This caused a mismatch with input config paths, preventing dependency tracking between permissions and their parent resources. With this fix, state and config paths are consistent.

## Tests
- Extended existing table-driven unit tests in `structwalk/`, `structaccess/`, and `structdiff/` with EmbeddedSlice cases
- Updated `acceptance/bundle/apps/job_permissions` to run on both terraform and direct engines
- Updated `acceptance/bundle/refschema` output to reflect corrected paths
- Updated direct engine plan output files for permissions path changes
