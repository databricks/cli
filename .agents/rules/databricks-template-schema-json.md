---
description: Databricks template schema file reference template and guidance
globs: **/databricks_template_schema.json
paths:
  - "**/databricks_template_schema.json"
---

# Databricks template schema files

A `databricks_template_schema.json` configures a bundle template's prompts. Model a new or changed file on the shipped templates in `libs/template/templates/*/databricks_template_schema.json` (`default-python` and `dbt-sql` cover most features). The supported keys beyond standard JSON schema (`order`, `pattern_match_failure_message`, `skip_prompt_if`, `format`, ...) are defined in `libs/jsonschema/extension.go` and `libs/jsonschema/schema.go`; template helpers such as `{{workspace_host}}`, `{{short_name}}` and `{{default_catalog}}` are in `libs/template/helpers.go`.

**RULE: Give each property a unique `order`, incrementing by 1.**
