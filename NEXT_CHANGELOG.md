# NEXT CHANGELOG

## Release v0.252.0

### Notable Changes

### Dependency updates
* Upgraded Go SDK to 0.69.0 ([#2867](https://github.com/databricks/cli/pull/2867))
* Upgraded to TF provider 1.79.0 ([#2869](https://github.com/databricks/cli/pull/2869))

### CLI

### Bundles
* Remove unused fields from resources.models schema: creation\_timestamp, last\_updated\_timestamp, latest\_versions and user\_id. Using them now raises a warning ([#2828](https://github.com/databricks/cli/pull/2828)).
* Preserve folder structure for app source code in bundle generate ([#2848](https://github.com/databricks/cli/pull/2848))
* Fix normalising requirements file path in dependencies section ([#2861](https://github.com/databricks/cli/pull/2861))
* Fix default-python template not to add environments when serverless=yes and include\_python=no ([#2866](https://github.com/databricks/cli/pull/2866))
* Fix handling of Unicode characters in Python support ([#2873](https://github.com/databricks/cli/pull/2873))
* Add support for secret scopes in DABs ([#2744](https://github.com/databricks/cli/pull/2744))
* Make `artifacts.*.type` optional in bundle JSON schema ([#2881](https://github.com/databricks/cli/pull/2881))
* Fix support for `spot_bid_max_price` field in Python support ([#2883](https://github.com/databricks/cli/pull/2883))

### API Changes
