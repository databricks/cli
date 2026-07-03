# NEXT CHANGELOG

## Release v1.7.0

### Notable Changes

### CLI

### Bundles

 * `bundle config-remote-sync` now derives its field filtering from the resource lifecycle metadata (`resources.yml`) instead of a hand-maintained table, so lifecycle rules added for the deploy plan automatically protect config sync ([#4677](https://github.com/databricks/cli/pull/4677)).
 * direct: policy-injected `custom_tags` and `cluster_log_conf` on job and standalone clusters no longer cause perpetual `bundle plan` updates when the field is absent from config; same for backend-computed `driver_node_type_flexibility`/`worker_node_type_flexibility` and `performance_target` ([#4677](https://github.com/databricks/cli/pull/4677)).

### Dependency updates

### API Changes
