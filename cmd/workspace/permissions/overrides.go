package permissions

import "github.com/spf13/cobra"

func cmdOverride(cmd *cobra.Command) {
	cmd.Long = `Permissions API are used to create read, write, edit, update and manage access
  for various users on different objects and endpoints.

  * **[Apps permissions](:service:apps)** — Manage which users can manage or
  use apps.

  * **[Cluster permissions](:service:clusters)** — Manage which users can
  manage, restart, or attach to clusters.

  * **[Cluster policy permissions](:service:clusterpolicies)** — Manage which
  users can use cluster policies.

  * **[Delta Live Tables pipeline permissions](:service:pipelines)** — Manage
  which users can view, manage, run, cancel, or own a Delta Live Tables
  pipeline.

  * **[Job permissions](:service:jobs)** — Manage which users can view,
  manage, trigger, cancel, or own a job.

  * **[MLflow experiment permissions](:service:experiments)** — Manage which
  users can read, edit, or manage MLflow experiments.

  * **[MLflow registered model permissions](:service:modelregistry)** — Manage
  which users can read, edit, or manage MLflow registered models.

  * **[Password permissions](:service:users)** — Manage which users can use
  password login when SSO is enabled.

  * **[Instance Pool permissions](:service:instancepools)** — Manage which
  users can manage or attach to pools.

  * **[Repo permissions](repos)** — Manage which users can read, run, edit, or
  manage a repo.

  * **[Serving endpoint permissions](:service:servingendpoints)** — Manage
  which users can view, query, or manage a serving endpoint.

  * **[SQL warehouse permissions](:service:warehouses)** — Manage which users
  can use or manage SQL warehouses.

  * **[Token permissions](:service:tokenmanagement)** — Manage which users can
  create or use tokens.

  * **[Workspace object permissions](:service:workspace)** — Manage which
  users can read, run, edit, or manage alerts, dbsql-dashboards, directories,
  files, notebooks and queries.

  For the mapping of the required permissions for specific actions or abilities
  and other important information, see [Access Control].

  Note that to manage access control on service principals, use **[Account
  Access Control Proxy](:service:accountaccesscontrolproxy)**.`
}

func init() {
	cmdOverrides = append(cmdOverrides, cmdOverride)
}
