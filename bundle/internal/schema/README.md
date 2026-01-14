# Bundle Schema Generator

This package generates the JSON schema for Databricks Asset Bundles configuration.

## Annotation Files

The schema generator uses three YAML files to add descriptions and metadata to the generated schema:

- **annotations_openapi.yml**: Auto-generated from the Databricks OpenAPI spec. Contains descriptions for SDK types (jobs, pipelines, etc.). Do not edit manually.

- **annotations_openapi_overrides.yml**: Manual overrides for OpenAPI annotations. Use this to fix or enhance descriptions from the OpenAPI spec without modifying the auto-generated file.

- **annotations.yml**: Manual annotations for CLI-specific types (e.g., `bundle`, `workspace`, `artifacts`). Missing annotations are automatically added with `PLACEHOLDER` descriptions.

## Annotation Priority

Files are merged in order, with later files taking precedence:
1. `annotations_openapi.yml` (base)
2. `annotations_openapi_overrides.yml` (overrides OpenAPI)
3. `annotations.yml` (CLI-specific, highest priority)

## Usage

Run `make schema` from the repository root to regenerate the bundle JSON schema.

To update OpenAPI annotations, set `DATABRICKS_OPENAPI_SPEC` to the path of the OpenAPI spec file before running.
