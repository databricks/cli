# NEXT CHANGELOG

## Release v0.300.0

### CLI

### Bundles
* engine/direct: Drop the deployment state entry on a recreate before the follow-up `Create`, so a `Create` failure no longer leaves a broken state with `invalid state: empty id` on the next `bundle plan` ([#5173](https://github.com/databricks/cli/pull/5173)).

### Dependency updates
