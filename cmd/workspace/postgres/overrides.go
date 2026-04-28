package postgres

import (
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
)

// createRoleOverride appends an example body to the auto-generated help.
// The --json flag binds to the inner Role object (CreateRoleRequest.Role,
// JSON-tagged "role"), so users supply spec/name/etc. directly. Without an
// example, the auto-generated `// TODO: complex arg: spec` flags leave no
// hint about the body shape and the API's "Field 'role' is required" error
// is unhelpful when the request body is wrong.
func createRoleOverride(createRoleCmd *cobra.Command, _ *postgres.CreateRoleRequest) {
	createRoleCmd.Long += `

  Body shape (passed via --json): fields go directly on the Role object.
  Do not wrap them in '{"role": ...}' — the CLI will strip the unknown
  outer key and the server will reject the empty body with "Field 'role'
  is required".

  Example — create a service-principal-backed role:

    databricks postgres create-role projects/<PROJECT_ID>/branches/<BRANCH_ID> \
      --role-id <SP_CLIENT_ID> \
      --json '{"spec": {"identity_type": "SERVICE_PRINCIPAL", "postgres_role": "<SP_CLIENT_ID>", "auth_method": "LAKEBASE_OAUTH_V1"}}'

  The example omits 'membership_roles' so the role starts with default
  privileges only — grant database/schema/table access separately via
  SQL, following least privilege. Set 'membership_roles' (e.g.
  ["DATABRICKS_SUPERUSER"]) only when broad administrative access is
  intentional.

  See databricks-sdk-go/service/postgres.RoleRoleSpec for the full set of
  spec fields.`
}

func init() {
	createRoleOverrides = append(createRoleOverrides, createRoleOverride)
}
