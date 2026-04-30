# NEXT CHANGELOG

## Release v0.300.0

### CLI

### Bundles

* Added `vector_search_indexes` as a first-class bundle resource on the direct deployment engine, alongside the existing `vector_search_endpoints`. Supports UC grants, drift detection (including out-of-band endpoint replacement that orphans an index), recreate-on-immutable-field-change, and asynchronous deletion waits. Recreating or deleting an index prompts for confirmation. Vector search has no Terraform provider, so this resource is direct-engine only ([#5123](https://github.com/databricks/cli/pull/5123)).

### Dependency updates
