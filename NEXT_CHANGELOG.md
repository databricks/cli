# NEXT CHANGELOG

## Release v0.295.0

### CLI

### Bundles

* **Breaking**: Nested variable references like `${var.foo_${var.tail}}` are now rejected with a warning and left unresolved. Previously the regex-based parser matched only the innermost `${var.tail}` by coincidence, which silently produced incorrect results. If you rely on dynamic variable name construction, use separate variables or target overrides instead ([#4747](https://github.com/databricks/cli/pull/4747)).

### Dependency updates

### API Changes
