# NEXT CHANGELOG

## Release v0.274.0

### Notable Changes

### CLI

### Dependency updates

### Bundles
* Fix a panic in TF when it fails to read the job ([#3799](https://github.com/databricks/cli/pull/3799))
* For secret scopes, no longer remove current user's permissions ([#3780](https://github.com/databricks/cli/pull/3780))
* Automatically add owner permissions during bundle initialization, this makes final permissions visible in 'bundle validate -o json' ([#3780](https://github.com/databricks/cli/pull/3780))
* Fix permissions for 'models' resource ([#3786](https://github.com/databricks/cli/pull/3786))

### API Changes
