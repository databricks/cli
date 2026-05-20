package postgres

import (
	"fmt"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
)

// createRoleOverride appends an example body to the auto-generated help and
// rejects wrapped {"role": ...} bodies with a clear client-side error.
// The --json flag binds to the inner Role object (CreateRoleRequest.Role,
// JSON-tagged "role"), so users supply spec/name/etc. directly. Without an
// example, the auto-generated `// TODO: complex arg: spec` flags leave no
// hint about the body shape and the API's "Field 'role' is required" error
// is unhelpful when the request body is wrong.
func createRoleOverride(createRoleCmd *cobra.Command, _ *postgres.CreateRoleRequest) {
	prevPreRunE := createRoleCmd.PreRunE
	createRoleCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := rejectWrappedRoleJSON(cmd); err != nil {
			return err
		}
		if prevPreRunE != nil {
			return prevPreRunE(cmd, args)
		}
		return nil
	}

	createRoleCmd.Long += `

  Body shape (passed via --json): fields go directly on the Role object.
  Do not wrap them in '{"role": ...}' — the CLI rejects wrapped bodies
  client-side with a hint pointing to the right shape.

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

// rejectWrappedRoleJSON returns a clear error when --json is a top-level
// object containing a "role" key. Without this guard the generated unmarshal
// strips the unknown outer "role" field with a warning and ships an empty
// body, and the server rejects with a confusing "Field 'role' is required"
// message.
func rejectWrappedRoleJSON(cmd *cobra.Command) error {
	// These checks are internal invariants — postgres create-role is a
	// generated command and always has a *flags.JsonFlag for --json. A
	// future codegen/refactor change could break that, and we want loud
	// breakage rather than a silently-disabled guard.
	flag := cmd.Flags().Lookup("json")
	if flag == nil {
		return fmt.Errorf("internal: postgres create-role expected a --json flag; this override is wired to the wrong command")
	}
	jf, ok := flag.Value.(*flags.JsonFlag)
	if !ok {
		return fmt.Errorf("internal: postgres create-role --json flag has unexpected type %T; expected *flags.JsonFlag", flag.Value)
	}
	return jf.RejectWrappedJSON("role", `databricks postgres create-role projects/<PROJECT_ID>/branches/<BRANCH_ID> \
    --role-id <SP_CLIENT_ID> \
    --json '{"spec": {"identity_type": "SERVICE_PRINCIPAL", "postgres_role": "<SP_CLIENT_ID>", "auth_method": "LAKEBASE_OAUTH_V1"}}'`)
}

func init() {
	createRoleOverrides = append(createRoleOverrides, createRoleOverride)
}
