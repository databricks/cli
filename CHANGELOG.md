# Version changelog

## 0.0.32
 * Add support for variables in bundle config. Introduces 4 ways of setting variable values, which in decreasing order of priority are: ([#383](https://github.com/databricks/cli/pull/383))([#359](https://github.com/databricks/cli/pull/359)).
	1. Command line flag. For example: `--var="foo=bar"`
	2. Environment variable. eg: BUNDLE_VAR_foo=bar
	3. Default value as defined in the applicable environments block
	4. Default value defined in variable definition
 * Make the git details bundle config block optional ([#372](https://github.com/databricks/cli/pull/372)).
 * Fix api post integration tests ([#371](https://github.com/databricks/cli/pull/371)).
 * Fix table of content by removing not required top-level item ([#366](https://github.com/databricks/cli/pull/366)).
 * Fix printing the tasks in job output in DAG execution order ([#377](https://github.com/databricks/cli/pull/377)).
 * Improved error message when 'bricks bundle run' is executed before 'bricks bundle deploy' ([#378](https://github.com/databricks/cli/pull/378)).

## 0.0.31

* Add OpenAPI command coverage (both workspace and account level APIs).

### Bundles

* Automatically populate a bundle's Git repository details in its configuration tree.

## 0.0.30

* Initial preview release of the Bricks CLI.
