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

Bundle templates pin the runtime versions they render. Most pins are macros in a template's `library/versions.tmpl` — `default` defines the most (DBR, DB Connect, serverless environment, Python, and the `databricks-bundles` package) — while `default-scala` defines its own in `library/template_variables.tmpl`, and some templates hardcode a version in the rendered file instead of going through a macro. The `.tmpl` files are the source of truth for the current values; the rules below describe how the pins relate, not what they are.

**RULE: Keep the serverless environment version, the Python pins, and the DB Connect pin mutually compatible. This is a hard constraint.** A serverless environment version dictates a runtime Python version, and the DB Connect pin must support that Python. Cross-check the [serverless environment version release notes](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/) and the [DB Connect requirements](https://docs.databricks.com/dev-tools/databricks-connect/python/index.html#requirements) whenever you change any of the three.

**RULE: Within that constraint, keep `conservative_db_connect_version_spec` as low as compatibility allows.** The DB Connect client is only forward-compatible (it reaches compute of its own version and higher), so the lowest compatible pin maximizes the range of DBR versions a customer can connect to; a higher pin rules out customers on older DBR, and they can move to a newer version themselves after initializing the template. Today the binding constraint is Python: the pin sits at 16.4 because that is the lowest DB Connect release whose Python (3.12) satisfies `python_version_spec` and the pinned serverless environment. Do not raise the pin merely to match a newer serverless environment version — raise it only when the compatibility constraint above forces it, or when the pinned DBR release falls out of support (see the [DBR release notes](https://docs.databricks.com/aws/en/release-notes/runtime/) for supported versions). See PR #3897 and PR #6378 for prior history.

**RULE: When bumping the serverless environment version, update every occurrence, not just the macros.** More than one template defines `serverless_environment_version` and those definitions must stay in sync, but several templates hardcode `environment_version` (and `requires-python`) in the files they render instead — grep both under `libs/template/templates/` and update every hit together.

**RULE: Do not sync a value across templates just because a macro name matches.** Outside the serverless environment version, a pin is a per-template choice that may legitimately differ, so check each template's intended value and change only the ones that should move.

Version pins are rendered into the materialized templates checked into `acceptance/`, so regenerate them with `./task test-update-templates` after changing one (see `.agents/rules/auto-generated-files.md`).
