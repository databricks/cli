# NEXT CHANGELOG

## Release v0.290.0

### CLI
* Add `completion install`, `uninstall`, and `status` subcommands ([#4581](https://github.com/databricks/cli/pull/4581))
* Wire profile name through CLI ToOAuthArgument for profile-based cache keys ([#4562](https://github.com/databricks/cli/pull/4562))
* Add host disambiguation and positional profile support to auth token ([#4574](https://github.com/databricks/cli/pull/4574))
* Update error messages to suggest 'databricks auth login' ([#4587](https://github.com/databricks/cli/pull/4587))
* Resolve --host to matching profile for token cache lookup ([#4591](https://github.com/databricks/cli/pull/4591))
* Improve auth token UX: profile selection and better empty-state handling ([#4584(https://github.com/databricks/cli/pull/4584)

### Bundles
* Added support for git_source and git_repository for Apps ([#4538](https://github.com/databricks/cli/pull/4538))

### Dependency updates
* Upgrade TF provider to 1.109.0 ([#4561](https://github.com/databricks/cli/pull/4561))
* Upgrade Go SDK to v0.110.0 ([#4552](https://github.com/databricks/cli/pull/4552))

### API Changes
* Bump databricks-sdk-go from v0.111.0 to v0.112.0.
