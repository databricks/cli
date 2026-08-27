---
description: Version pins in bundle template versions.tmpl files (DBR, DB Connect, serverless env, Python)
globs:
  - "libs/template/templates/**/library/versions.tmpl"
paths:
  - "libs/template/templates/**/library/versions.tmpl"
---

# Bundle template version pins

Each bundle template pins the runtime versions a freshly initialized project
ships with in `library/versions.tmpl`. The `default` template pins the full set
— `latest_lts_dbr_version`, `conservative_db_connect_version_spec`,
`serverless_environment_version`, `python_version_spec`, and
`default_python_version`; the SQL templates (`dbt-sql`, `default-sql`) define
only a subset.

**RULE: Keep `conservative_db_connect_version_spec` (in `default/`) at the lowest version that still works. Bump it only when the pinned DBR release falls out of support — never just to match the newest serverless environment version.** The DB Connect client is only forward-compatible: it reaches compute of its own version and higher. The lowest working pin therefore maximizes the range of DBR versions a customer can connect to, whereas a high pin rules out customers on older DBR. Customers can upgrade themselves after initializing the template. This is why the pin typically lags the newest release. See PR #3897 and PR #6378 for prior history.

**RULE: In `default/`, keep `serverless_environment_version`, the Python pins (`python_version_spec` / `default_python_version`), and `conservative_db_connect_version_spec` mutually compatible.** They form one set: a serverless environment version dictates a runtime Python version, and the DB Connect pin must support that Python. For example, environment version 5 uses Python 3.12, and DB Connect 16.4 supports Python 3.12. When you change any one, cross-check the other two against the [serverless environment version release notes](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/) and the [DB Connect requirements](https://docs.databricks.com/dev-tools/databricks-connect/python/index.html#requirements).

**RULE: `serverless_environment_version` is the one macro that must stay in sync across templates — it is pinned to the same value (`5`) in both `default/` and `dbt-sql/`, so bump it in both.** Other macros that appear in more than one template hold intentionally different values and must NOT be synced: `latest_lts_dbr_version` is `16.4` in `default/` but `15.4` in the SQL templates, and each SQL template pins its own `latest_lts_db_connect_version_spec` (a distinct macro from `default/`'s `conservative_db_connect_version_spec`). Change only the templates that define a given macro, and only for the same reason.

Changing a version pin changes rendered template output, so regenerate the acceptance goldens afterward (see [auto-generated-files.md](auto-generated-files.md)).
