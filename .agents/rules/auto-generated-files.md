---
description: Rules for how to deal with auto-generated files
globs:
  - ".codegen/cli.json"
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
  - "bundle/direct/dresources/configs/*.generated.yml"
  - "bundle/internal/schema/annotations.generated.yml"
  - "bundle/internal/validation/generated/*.go"
  - "bundle/schema/jsonschema.json"
  - "python/databricks/bundles/version.py"
  - "python/databricks/bundles/*/__init__.py"
  - "python/databricks/bundles/*/_models/*.py"
paths:
  - ".codegen/cli.json"
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
  - "bundle/direct/dresources/configs/*.generated.yml"
  - "bundle/internal/schema/annotations.generated.yml"
  - "bundle/internal/validation/generated/*.go"
  - "bundle/schema/jsonschema.json"
  - "python/databricks/bundles/version.py"
  - "python/databricks/bundles/*/__init__.py"
  - "python/databricks/bundles/*/_models/*.py"
---

# Auto-generated files

Files matching this rule's globs are generated; they usually say so in a header comment. `Taskfile.yml` (`generate*` and `test-update*` tasks) shows what produces what.

**RULE: Do not manually edit generated files. Edit the source (logic, template, annotation, test script) and regenerate, then commit both.**

- `./task generate` — runs all generators below.
- `./task generate-cligen` — `cmd/account/**`, `cmd/workspace/**` and `.gitattributes` from the checked-in `.codegen/cli.json`.
- `./task generate-clijson` — refreshes `.codegen/cli.json` and `internal/genkit/tagging.py` from the OpenAPI spec (requires the universe repo).
- `./task generate-direct` — direct engine generated YAML.
- `./task generate-schema` — `bundle/schema/jsonschema.json` (upstream docs come from `.codegen/cli.json`). It also normalizes the hand-written `bundle/internal/schema/annotations.yml` (docs for fields only the CLI defines) and `annotations.overrides.yml` (entries for fields `cli.json` already defines, each a candidate to fix upstream and delete) in place, routing entries between them by whether the field exists upstream and dropping entries for fields that no longer exist. It fully rewrites `bundle/internal/schema/annotations.generated.yml`, which lists the fields with no documentation anywhere as `PLACEHOLDER` entries. To document one, add the entry to `annotations.yml` and regenerate.
- `./task generate-validation` — `bundle/internal/validation/generated/`.
- `./task pydabs-codegen` — Python bundle models.
- Mocks: see `internal/mocks/README.md`.
- Acceptance outputs: `./task test-update` (`./task test-update-templates` for `acceptance/bundle/templates` only). See `.agents/rules/testing.md`.
