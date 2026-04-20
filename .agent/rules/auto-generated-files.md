---
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
paths:
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

# Rules for how to deal with auto-generated files

## Identification

Files matching this rule's glob pattern are most likely generated artifacts. Auto-generated files generally have a comment (when the file type allows it) at or near the top indicating they are generated, or their name or path signals it. You may also consult the Makefile as a starting point to determine if a file is auto-generated.

## Rules

**RULE: Do not manually edit auto-generated files.**

**RULE: To change a generated file, edit the source and regenerate.**

1. Find the source logic, template, or annotation that drives the file.
2. Run the appropriate generator or update command.
3. Commit both the source change (if any) and the regenerated outputs.

### Core generation commands

- Everything, in one shot:
  - `task generate` — aggregator that runs all generators below
- OpenAPI SDK/CLI command stubs and related generated artifacts:
  - `task generate:genkit`
  - Includes generated `cmd/account/**`, `cmd/workspace/**`, `.gitattributes`, `internal/genkit/tagging.py`.
- Direct engine generated YAML:
  - `task generate:direct` (or `task generate:direct:apitypes`, `task generate:direct:resources`)
- Bundle schemas:
  - `task generate:schema`
  - `task generate:schema-docs`
  - This can also refresh `bundle/internal/schema/annotations_openapi.yml` when OpenAPI annotation extraction is enabled.
- Bundle docs:
  - `task generate:docs`
- Validation generated code:
  - `task generate:validation`
- Mock files:
  - `go run github.com/vektra/mockery/v2@b9df18e0f7b94f0bc11af3f379c8a9aea1e1e8da`
- Python bundle codegen:
  - `task python:codegen`

### Acceptance and test generated outputs

**RULE: Do not hand-edit acceptance outputs.** Exception: rare, intentional mass replacement when explicitly justified by repo guidance.

Regeneration commands:

- `make test-update`
- `make test-update-templates` (templates only)
- `make generate-out-test-toml` (only `out.test.toml`)

Typical generated files:

- `acceptance/**/out*`
- `acceptance/**/output.txt`
- `acceptance/**/output.*.txt`
- `acceptance/**/output/**` (materialized template output trees)

**RULE: When touching acceptance sources, regenerate outputs instead of editing generated files.** Sources include `databricks.yml`, scripts, templates, and test config.
