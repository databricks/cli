Configuration
---

There are two main configuration concepts of `bricks` CLI: per-project `databricks.yml` and per-machine `~/.databrickscfg` file.

## `~/.databrickscfg`

The purpose of this file is hold connectivity profiles with possibly clear-text credentials to Databricks Workspaces or Databricks Accounts. Almost all entries from this configuration file can be set through environment variables. The same configuration file can be read via the official Databricks GoLang SDK and Databricks Python SDK. Legacy Databricks CLI supports reading only `host`, `token`, `username`, and `password` configuration options.

 * `host` _(string)_: Databricks host for either workspace endpoint or Accounts endpoint. Environment: `DATABRICKS_HOST`.
 * `account_id` _(string)_: Databricks Account ID for Accounts endpoint. Environment: `DATABRICKS_ACCOUNT_ID`.
 * `token` _(string)_: Personal Access Token (PAT). Environment: `DATABRICKS_TOKEN`.
 * `username` _(string)_: Username part of basic authentication. Environment: `DATABRICKS_USERNAME`.
 * `password` _(string)_: Password part of basic authentication. Environment: `DATABRICKS_PASSWORD`.
 * `profile` _(string)_: Connection profile specified within `~/.databrickscfg`. Environment: `DATABRICKS_CONFIG_PROFILE`.
 * `config_file` _(string)_: Location of the Databricks CLI credentials file. By default, it is located in `~/.databrickscfg`. Environment: `DATABRICKS_CONFIG_FILE`.
 * `google_service_account` _(string)_: Google Compute Platform (GCP) Service Account e-mail used for impersonation in the Default Application Credentials Flow that does not require a password. Environment: `DATABRICKS_GOOGLE_SERVICE_ACCOUNT`
 * `google_credentials` _(string)_: GCP Service Account Credentials JSON or the location of these credentials on the local filesustem.  Environment: `GOOGLE_CREDENTIALS`.
 * `azure_workspace_resource_id` _(string)_: Azure Resource Manager ID for Azure Databricks workspace, which is exhanged for a Host. Environment: `DATABRICKS_AZURE_RESOURCE_ID`.
 * `azure_use_msi` _(string)_: Instruct to use Azure Managed Service Identity passwordless authentication flow for Service Principals. Environment: `ARM_USE_MSI`.
 * `azure_client_secret` _(string)_: Azure Active Directory Service Principal secret. Environment: `ARM_CLIENT_SECRET`.
 * `azure_client_id` _(string)_: Azure Active Directory Service Principal Application ID. Environment: `ARM_CLIENT_ID`
 * `azure_tenant_id` _(string)_: Azure Active Directory Tenant ID. Environment: `ARM_TENANT_ID`
 * `azure_environment` _(string)_: Azure Environment (Public, UsGov, China, Germany) has specific set of API endpoints. Defaults to `PUBLIC`. Environment: `ARM_ENVIRONMENT`.
 * `auth_type` _(string)_: When multiple auth attributes are available in the environment, use the auth type specified by this argument. This argument also holds currently selected auth.
* `http_timeout_seconds` _(int)_: Number of seconds for HTTP timeout.
* `debug_truncate_bytes` _(int)_: Truncate JSON fields in debug logs above this limit. Default is 96. Environment: `DATABRICKS_DEBUG_TRUNCATE_BYTES`
* `debug_headers` _(bool)_: Debug HTTP headers of requests made by the application. Default is false, as headers contain sensitive data, like tokens. Environment: `DATABRICKS_DEBUG_HEADERS`.
 * `rate_limit` _(int)_: Maximum number of requests per second made to Databricks REST API. Environment: `DATABRICKS_RATE_LIMIT`

## `databricks.yml`

Frequently, developers work on more than a single project from their workstations. Having a per-project `databricks.yml` configuration file created by [`bricks init`](project-lifecycle.md#init) helps achieving resource isolation and connectivity credentials flexibility. 

### Development Cluster

Every project, with the exception of several [project flavors](project-flavors.md) may have a Databricks Cluster, where groups or individual data engineers run Spark queries in the Databricks Runtime. It's also possible to [isolate](#isolation-levels) clusters.

### Project Flavor

[Project Flavors](project-flavors.md) are the features, that detect the intended behavior of deployments during the [project lifecycle](project-lifecycle.md).

### Isolation Levels

It's possible to achieve _soft isolation_ levels for multiple developers to independently work on the same project, like having different branches in Git.