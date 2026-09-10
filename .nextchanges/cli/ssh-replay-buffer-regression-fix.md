* Fix SSH tunnel regression where continuous transfers >1 MiB crash the session by degrading to non-resumable mode when the replay buffer fills. ([#6609](https://github.com/databricks/cli/pull/6609))
