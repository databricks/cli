---
name: auto-generated-files
description: Rules for how to deal with auto-generated files
globs:
  - ".codegen/openapi.json"
  - ".gitattributes"
  - "acceptance/**/out*"
  - "acceptance/**/output.txt"
  - "acceptance/**/output.*.txt"
  - "acceptance/**/output/**"
  - "cmd/account/*.go"
  - "cmd/account/**/*.go"
  - "cmd/workspace/*.go"
  - "cmd/workspace/**/*.go"
  - "internal/genkit/tagging.py"
  - "internal/mocks/**/*.go"
  - "bundle/direct/dresources/*.generated.yml"
  - "bundle/docsgen/output/**/*.md"
  - "bundle/internal/schema/annotations_openapi.yml"
  - "bundle/internal/validation/generated/*.go"
  - "bundle/schema/jsonschema.json"
  - "bundle/schema/jsonschema_for_docs.json"
  - "python/databricks/bundles/version.py"
  - "python/databricks/bundles/*/__init__.py"
  - "python/databricks/bundles/*/_models/*.py"
---

# Auto-generated files

## Identification

The files matching this rule glob pattern are most likely generated artifacts. Auto-generated files generally have a comment (if the file type allows for comments) at or near the top of the file indicating that they are generated, or their file name/path may indicate they are generated. You may also consult Makefile as starting point to determine if a file is auto-generated.

## Rules

DO NOT "MANUALLY" EDIT THESE FILES!

If a change is needed in any matched file:
1. Find the source logic/template/annotation that drives the file.
2. Run the appropriate generator/update command.
3. Commit both the source change (if any) and regenerated outputs.

### Core generation commands

- OpenAPI SDK/CLI command stubs and related generated artifacts:
  - `make generate`
  - Includes generated `cmd/account/**`, `cmd/workspace/**`, `.gitattributes`, `internal/genkit/tagging.py`, and direct engine refresh.
- Direct engine generated YAML:
  - `make generate-direct` (or `make generate-direct-apitypes`, `make generate-direct-resources`)
- Bundle schemas:
  - `make schema`
  - `make schema-for-docs`
  - This can also refresh `bundle/internal/schema/annotations_openapi.yml` when OpenAPI annotation extraction is enabled.
- Bundle docs:
  - `make docs`
- Validation generated code:
  - `make generate-validation`
- Mock files:
  - `go run github.com/vektra/mockery/v2@b9df18e0f7b94f0bc11af3f379c8a9aea1e1e8da`
- Python bundle codegen:
  - `make -C python codegen`

### Acceptance and test generated outputs

Acceptance outputs are generated and should not be hand-edited (except rare, intentional mass replacement when explicitly justified by repo guidance).

- Preferred regeneration:
  - `make test-update`
  - `make test-update-templates` (templates only)
  - `make generate-out-test-toml` (only `out.test.toml`)
- Typical generated files include:
  - `acceptance/**/out*`
  - `acceptance/**/output.txt`
  - `acceptance/**/output.*.txt`
  - `acceptance/**/output/**` (materialized template output trees)

When touching acceptance sources (`databricks.yml`, scripts, templates, or test config), regenerate outputs instead of editing generated files directly.
