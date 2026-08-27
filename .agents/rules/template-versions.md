---
description: Version pins in bundle template versions.tmpl files (DBR, DB Connect, serverless env, Python)
globs:
  - "libs/template/templates/**/library/versions.tmpl"
paths:
  - "libs/template/templates/**/library/versions.tmpl"
---

# Bundle template version pins

Several bundle templates pin their runtime versions in `library/versions.tmpl`: `default` pins the most (DBR, DB Connect, serverless environment, Python, and the `databricks-bundles` package), while `dbt-sql` and `default-sql` pin subsets. The `.tmpl` files are the source of truth for the current values — the rules below describe how the pins relate, not what they are. (`default-scala` pins its versions in `library/template_variables.tmpl` instead and is out of scope here.)

**RULE: Keep the serverless environment version, the Python pins, and the DB Connect pin mutually compatible. This is a hard constraint.** A serverless environment version dictates a runtime Python version, and the DB Connect pin must support that Python. Cross-check the [serverless environment version release notes](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/) and the [DB Connect requirements](https://docs.databricks.com/dev-tools/databricks-connect/python/index.html#requirements) whenever you change any of the three.

**RULE: Within that constraint, keep `conservative_db_connect_version_spec` as low as compatibility allows.** The DB Connect client is only forward-compatible (it reaches compute of its own version and higher), so the lowest compatible pin maximizes the range of DBR versions a customer can connect to; a higher pin rules out customers on older DBR. Do not raise it merely to match the newest serverless environment — raise it only when compatibility forces it, or when the pinned DBR release falls out of support (see the [DBR release notes](https://docs.databricks.com/aws/en/release-notes/runtime/) for supported versions). Customers can move to a newer version themselves after initializing the template. See PR #3897 and PR #6378 for prior history.

**RULE: `serverless_environment_version` must stay in sync across every template that defines it — bump them together.** Other macros are defined per-template and may legitimately differ (the SQL templates, for example, currently pin an older `latest_lts_dbr_version` than `default`). Do not sync a value across templates just because a macro name matches: check each template's intended value and change only the ones that should move, for the same reason.

Changing a version pin changes rendered template output, so regenerate the acceptance goldens afterward (see [auto-generated-files.md](auto-generated-files.md)).
