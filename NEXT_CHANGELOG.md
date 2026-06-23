# NEXT CHANGELOG

## Release v1.5.0

### Notable Changes

### CLI
* `databricks ssh connect` now opens an interactive `bash` login shell by default instead of the compute image's default `/bin/sh`, falling back gracefully when `bash` is unavailable. Passing an explicit remote command (`-- <cmd>`) is unaffected ([#5687](https://github.com/databricks/cli/pull/5687)).

* `databricks ssh connect` interactive sessions now start in the user's workspace home folder (`/Workspace/Users/<email>`) instead of the OS home directory, falling back to the OS home when that folder is unavailable ([#5688](https://github.com/databricks/cli/pull/5688)).

### Bundles
* `bundle run` now prints the modern job run URL (`/jobs/<id>/runs/<id>`) so that non-admin users permitted to view the run are taken to the run instead of the workspace homepage.
* Fix missing field descriptions in the bundle JSON schema for fields whose upstream API docs arrived after the field was first annotated (e.g. `vector_search_endpoints.*.target_qps`); stale placeholder markers no longer hide them ([#5588](https://github.com/databricks/cli/pull/5588)).
* Fix `bundle deploy --plan` dropping a `postgres_role`'s `role_id`, which caused the role to be recreated on the next deploy ([#5672](https://github.com/databricks/cli/pull/5672)).

### Dependency updates
* Bump `github.com/databricks/databricks-sdk-go` from v0.141.0 to v0.147.0 ([#5636](https://github.com/databricks/cli/pull/5636)).
* Bump Terraform provider from v1.117.0 to v1.118.0 ([#5637](https://github.com/databricks/cli/pull/5637)).

### API Changes
