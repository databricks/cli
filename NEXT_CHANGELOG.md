# NEXT CHANGELOG

## Release v1.5.0

### Notable Changes

### CLI
* `workspace export-dir` no longer aborts when a workspace object's name is not a legal local filename (e.g. a notebook named `New Notebook 2026-05-04 13:54:24` whose `:` is illegal on Windows). Such files are now exported under a sanitized name with a warning and the export completes ([#5171](https://github.com/databricks/cli/issues/5171)).
* `ssh connect` now opens an interactive `bash` login shell by default instead of the compute image's default `/bin/sh`, falling back gracefully when `bash` is unavailable. Passing an explicit remote command (`-- <cmd>`) is unaffected ([#5687](https://github.com/databricks/cli/pull/5687)).
* `ssh connect` interactive sessions now start in the user's workspace home folder (`/Workspace/Users/<email>`) instead of the OS home directory, falling back to the OS home when that folder is unavailable ([#5688](https://github.com/databricks/cli/pull/5688)).

### Bundles
* Add documentation for the common bundle resource fields `permissions`, `lifecycle`, and `grants` in the JSON schema, so they surface in editor completions and the docs.
* `bundle run` now prints the modern job run URL (`/jobs/<id>/runs/<id>`) so that non-admin users permitted to view the run are taken to the run instead of the workspace homepage.
* References to a registered model's `registered_model_id` now resolve under the direct engine, matching Terraform behavior ([#5621](https://github.com/databricks/cli/pull/5621)).
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them ([#5588](https://github.com/databricks/cli/pull/5588)).
* Fix `bundle deploy --plan` dropping a `postgres_role`'s `role_id`, which caused the role to be recreated on the next deploy ([#5672](https://github.com/databricks/cli/pull/5672)).
* direct: Fix spurious cluster recreate when `apply_policy_default_values: true` is set ([#5693](https://github.com/databricks/cli/pull/5693)).

### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.141.0 to v0.147.0 ([#5636](https://github.com/databricks/cli/pull/5636)).
* Bump Terraform provider from v1.117.0 to v1.118.0 ([#5637](https://github.com/databricks/cli/pull/5637)).

### API Changes
