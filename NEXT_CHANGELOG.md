# NEXT CHANGELOG

## Release v0.250.0

### Notable Changes
* Added inline script execution support to bundle run. You can now run scripts in the same authentication context as a DAB using the databricks bundle run command. ([#2413](https://github.com/databricks/cli/pull/2413))

### Dependency updates
* Upgrade TF provider to 1.75.0 ([#2775](https://github.com/databricks/cli/pull/2775))

### CLI
* Added `databricks apps run-local` command to run Databricks apps locally ([#2555](https://github.com/databricks/cli/pull/2555))

### Bundles
* Raise an error when Unity Catalog volumes are used for paths other than artifacts ([#2754](https://github.com/databricks/cli/pull/2754))
* Fixed issue with jobs and pipelines declared in Python not showing in "Bundle resource explorer" in VSCode ([#2764](https://github.com/databricks/cli/pull/2764))
* Made `experimental/python/mutators` and `experimental/python/resources` fields optional in JSON schema ([#2761](https://github.com/databricks/cli/pull/2761))

### API Changes
