# Test path translation (nominal behavior)

As of v0.216.0 (PR at https://github.com/databricks/cli/pull/1273), all paths in a resource
definition are resolved relative to the directory of the file where they are defined.

This is more intuitive than the previous behavior (see `../fallback/README.md` for details).
