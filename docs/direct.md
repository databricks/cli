## Status

Status: experimental, only recommended for development targets or evaluation, not for production use.
Known issues: https://github.com/databricks/cli/issues?q=state%3Aopen%20label%3Aengine%2Fdirect

## Reporting bugs

Click "new issue" on [https://github.com/databricks/cli/issues](https://github.com/databricks/cli/issues) and select "Bug report for Databricks Asset Bundles with direct deployment engine".

Please ensure you are on the latest version of CLI when reporting an issue.

## Background

Databricks Asset Bundles were originally implemented on top of databricks-terraform-provider.
Since version 0.279.0 DABs support two different deployment engines: "terraform" and "direct".
The latter does not make use of Terraform.
It is intended to be a drop-in replacement and will become the only engine we support in 2026\.

## Advantages

The new engine implements resources CRUD directly on top of SDK Go and provides the following benefits over the original one:

* Self-contained binary that does not require downloading Terraform and terraform-provider-databricks before deployment.
  * Avoid issues with firewalls/proxies/custom provider registries.
* Explanation why a given action is planned and detailed diff of changes available in "bundle plan \-o json".
* Faster deployment.
* Simplified development of new resources, implement CRUD directly in CLI repo, no need to coordinate with terraform provider release.

## Disadvantages

There are known issues, see https://github.com/databricks/cli/issues?q=state%3Aopen%20label%3Aengine%2Fdirect

## Usage

### Migrating the existing deployment

The direct engine uses its own state file, also JSON, but with a different schema from terraform state file. In order to migrate an existing Terraform-based deployment, use the "bundle deployment migrate" command. The command reads IDs from the existing deployment.

The full sequence of operations:

1. Perform full deployment with Terraform: bundle deploy \-t my\_target
2. Migrate state file locally: bundle deployment migrate \-t my\_target
3. Verify that migration was successful: bundle plan should work and should not show any changes to be planned: bundle plan \-t my\_target
4. If not satisfied with the result, remove new state file: rm .databricks/bundle/my\_target/resources.json
5. If satisfied with the result, do a deployment to synchronize the state file to the workspace: bundle deploy \-t my\_target

### Using on new bundles

For bundles that were never deployed, the migrate command will not work. Instead, deploy with an environment variable set: DATABRICKS\_BUNDLE\_ENGINE=direct bundle deploy \-t my\_target.

## Differences from terraform

### Diff calculation

Unlike Terraform which maintains a single "resource state" (a mix of local configuration and remote state), the new engine keeps these separate and only records local configuration in its state file.

The diff calculation is done in 2 steps:

* Step 1: local changes: local config is compared to the snapshot of config used for the most recent deployment. The remote state plays no role there.
* Step 2: remote changes: remote state is compared to the snapshot of config used for the most recent deployment.

The consequences of this are:

* Changes in databricks.yml resource configuration are never ignored and will always trigger an update.
* Resource fields behaving "inconsistently" and not handled by the implementation do not trigger ["Provider produced inconsistent result after apply"](https://github.com/databricks/terraform-provider-databricks/issues?q=is%3Aissue%20%22Provider%20produced%20inconsistent%20result%20after%20apply%22) error. Such resources will be deployed successfully by direct engine. This can result in a drift: the deployed resources will be marked as "to be updated" during the next plan/deploy.

### $resources references lookup

Note, the most common use of $resources is resolving ID ($resources.jobs.foo.id) which should behave identically between terraform and direct engine. The differences can happen in other fields.

The resolution of $resources references in direct engine (e.g. $resources.pipelines.foo.name) is performed in two steps:

1. References pointing to fields that are present in the config ("local") are resolved during the plan phase to the value provided in the local config.
2. References that are not present in the config, are resolved from remote state, that is state fetched via appropriate GET request for a given resource.

The schema that is used for $resource resolution is available in [acceptance/bundle/refschema/out.fields.txt](https://github.com/databricks/cli/blob/main/acceptance/bundle/refschema/out.fields.txt). The fields marked as "ALL" and "STATE" can be used for local resolution. The fields marked as "ALL" or "REMOTE" can be used for remote resolution.
