# Include resource in auto-generation

Checklist:
- Register the resource in `bundle/config/resources.go`.
- Add or update resource config types under `bundle/config/resources/`.
- Regenerate generated artifacts using the relevant make targets:
  - `make schema` (optionally with `DATABRICKS_OPENAPI_SPEC=...`)
  - `make schema-for-docs`
  - `make generate-validation`
  - `make generate-direct`
- If schema descriptions need overrides, update:
  - `bundle/internal/schema/annotations_openapi_overrides.yml`
  - `bundle/internal/schema/annotations.yml`

Keep this file as a run-order checklist. Canonical docs for generation-related behavior live in:
- `bundle/docsgen/README.md`
- `bundle/internal/tf/codegen/README.md`
