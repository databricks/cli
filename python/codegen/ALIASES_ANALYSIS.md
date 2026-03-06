# Python Codegen: Backward Compatibility Aliases Analysis

## Context

The `Privilege` and `PrivilegeAssignment` types are duplicated across three namespaces
(`catalogs`, `schemas`, `volumes`) by design — each namespace is self-contained with no
cross-namespace imports, avoiding coupling issues from Databricks SDK v1.

Before this change, each namespace had resource-specific grant types:
- `catalogs`: `CatalogGrant`, `CatalogGrantPrivilege`
- `schemas`: `SchemaGrant`, `SchemaGrantPrivilege`
- `volumes`: `VolumeGrant`, `VolumeGrantPrivilege`

These were replaced by the shared (but duplicated) `PrivilegeAssignment` and `Privilege`.

## Alias Mapping

| namespace | old name | new name |
|---|---|---|
| catalogs | `CatalogGrant` | `PrivilegeAssignment` |
| catalogs | `CatalogGrantDict` | `PrivilegeAssignmentDict` |
| catalogs | `CatalogGrantParam` | `PrivilegeAssignmentParam` |
| catalogs | `CatalogGrantPrivilege` | `Privilege` |
| catalogs | `CatalogGrantPrivilegeParam` | `PrivilegeParam` |
| schemas | `SchemaGrant` | `PrivilegeAssignment` |
| schemas | `SchemaGrantDict` | `PrivilegeAssignmentDict` |
| schemas | `SchemaGrantParam` | `PrivilegeAssignmentParam` |
| schemas | `SchemaGrantPrivilege` | `Privilege` |
| schemas | `SchemaGrantPrivilegeParam` | `PrivilegeParam` |
| volumes | `VolumeGrant` | `PrivilegeAssignment` |
| volumes | `VolumeGrantDict` | `PrivilegeAssignmentDict` |
| volumes | `VolumeGrantParam` | `PrivilegeAssignmentParam` |
| volumes | `VolumeGrantPrivilege` | `Privilege` |
| volumes | `VolumeGrantPrivilegeParam` | `PrivilegeParam` |

## Options Considered

### Option A: Append aliases to `__init__.py` only (CHOSEN)

Modify `_write_exports()` to read an `ALIASES` dict and emit `OldName = NewName`
assignments after imports, plus add old names to `__all__`.

- Pros: minimal change, no new files, fits existing pattern
- Cons: only `from databricks.bundles.catalogs import CatalogGrant` works,
  not deep `_models`-level imports
- Sufficient for all realistic use cases

### Option B: Generate a separate `_models/aliases.py` per namespace

Produces a separate file with imports + alias assignments.

- Pros: supports both package-level and `_models`-level imports
- Cons: more files, more generator complexity

### Option C: Append aliases to the target model file

Appends `CatalogGrant = PrivilegeAssignment` to `privilege_assignment.py`.

- Cons: pollutes generated model files with namespace-specific legacy names

## Implementation (Option A)

1. New file: `codegen/aliases_patch.py` — defines `ALIASES: dict[str, dict[str, str]]`
2. `main.py` `_write_exports()` — append alias assignments and include old names in `__all__`
