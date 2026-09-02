---
description: Version pins in bundle templates (DBR, DB Connect, serverless env, Python)
globs:
  - "libs/template/templates/**/library/versions.tmpl"
  - "libs/template/templates/**/library/template_variables.tmpl"
  - "libs/template/templates/**/resources/*.job.yml.tmpl"
  - "libs/template/templates/**/pyproject.toml.tmpl"
paths:
  - "libs/template/templates/**/library/versions.tmpl"
  - "libs/template/templates/**/library/template_variables.tmpl"
  - "libs/template/templates/**/resources/*.job.yml.tmpl"
  - "libs/template/templates/**/pyproject.toml.tmpl"
---

# Bundle template version pins

Bundle templates pin the runtime versions they render. Most pins are macros in `libs/template/templates/default/library/versions.tmpl` (DBR, DB Connect, serverless environment, Python, and the `databricks-bundles` package), and that single file serves five shipped templates: `default-minimal`, `default-python`, `lakeflow-pipelines` and `pydabs` all alias it through `"template_dir": "../default"` in their `databricks_template_schema.json`. `dbt-sql` defines its own subset, `default-scala` defines its own in `library/template_variables.tmpl`, and two templates hardcode a version in the files they render instead of going through a macro. The `.tmpl` files are the source of truth for the current values; the rules below describe how the pins relate, not what they are.

**RULE: Keep the serverless environment version, the Python pins, the DB Connect pin, and `latest_lts_dbr_version` mutually compatible. This is a hard constraint.** A serverless environment version dictates a runtime Python version, and the DB Connect pin must support that Python. The DBR LTS pin belongs in the same check rather than being treated as independent — the two have drifted before: `latest_lts_dbr_version` reached 16.4 in #3671 while the DB Connect pin stayed on 15.4 until #6378. Cross-check the [serverless environment version release notes](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/) and the [DB Connect requirements](https://docs.databricks.com/dev-tools/databricks-connect/python/index.html#requirements) whenever you change any of the four.

**RULE: Within that constraint, keep `default`'s `conservative_db_connect_version_spec` as low as compatibility allows.** The DB Connect client is only forward-compatible (it reaches compute of its own version and higher), so the lowest compatible pin maximizes the range of DBR versions a customer can connect to; a higher pin rules out customers on older DBR, and they can move to a newer version themselves after initializing the template. The floor is the lowest release that is *both* still supported — in practice the current DBR LTS, since the non-LTS minor releases below it fall out of support first — *and* whose Python satisfies `python_version_spec`. Python alone does not determine the floor: a whole DBR major line typically shares one Python version, so several releases below the LTS will also satisfy the Python pin. Keep the upper bound at the next minor release, so the spec stays inside one minor: widening it lets a freshly initialized project resolve a client far newer than the floor advertises, which is the opposite of conservative. Do not raise the pin merely to match a newer serverless environment version — raise it only when the constraint above forces it, or when the pinned release falls out of support (see the [DBR release notes](https://docs.databricks.com/aws/en/release-notes/runtime/) for supported versions). See PR #3897 and PR #6378 for prior history.

This rule governs `default`'s pin. `default-scala` deliberately does the opposite: it tracks the newest LTS in `dbr_version` and derives its client from that macro in `template/{{.project_name}}/build.sbt.tmpl` as an open-ended `.+` dependency. Do not lower `dbr_version`, or pin that dependency, to satisfy the rule above.

**RULE: When bumping the serverless environment version or the Python version, update every occurrence, not just the macros.** More than one template defines `serverless_environment_version`, and those definitions must stay in sync. Two templates hardcode the value in the files they render instead — `default-scala` in one job and `lakeflow-integrations` in two — and `lakeflow-integrations` also hardcodes `requires-python`. A Python bump additionally needs `default_python_version`, which supplies the notebook kernel version in `src/sample_notebook.ipynb.tmpl`. Grep `environment_version`, `requires-python` and `default_python_version` under `libs/template/templates/` and move every hit together.

**RULE: Do not sync a value across templates just because a macro name matches.** Outside the serverless environment version, a pin is a per-template choice that may legitimately differ, so check each template's intended value and change only the ones that should move.

Version pins are rendered into the materialized templates checked into `acceptance/`, and not all of those live under `acceptance/bundle/templates/` — `acceptance/pipelines/` renders the pins too. Regenerate with `./task test-update`, not `./task test-update-templates`, which covers only the `bundle/templates` subtree. See `.agents/rules/testing.md` and `.agents/rules/auto-generated-files.md`.
