// Postgres Role resource for the direct deployment engine.
//
// Terraform resource: databricks_postgres_role
//
//	https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/postgres_role
//
// REST API: Lakebase Postgres Roles
//
//	https://docs.databricks.com/api/workspace/postgres/createrole
package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresRole struct {
	client *databricks.WorkspaceClient
}

// PostgresRoleState keeps role_id and parent as separate fields rather than a
// pre-joined hierarchical name. That alignment matters because bundle variable
// resolution only rewrites state fields whose JSON paths appear in the input
// config's refs map (parent, role_id, etc.); a synthesized "name" field built
// from input.Parent at PrepareState time would keep the literal ${...} string
// when parent comes from a resource reference.
type PostgresRoleState struct {
	postgres.RoleRoleSpec

	// RoleId is the leaf id, matching the user-facing config.
	RoleId string `json:"role_id"`

	// Parent is "projects/{project_id}/branches/{branch_id}".
	Parent string `json:"parent"`
}

// Custom marshaler needed because embedded RoleRoleSpec has its own
// MarshalJSON which would otherwise take over and ignore the additional fields.
func (s *PostgresRoleState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresRoleState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (*ResourcePostgresRole) New(client *databricks.WorkspaceClient) *ResourcePostgresRole {
	return &ResourcePostgresRole{client: client}
}

func (*ResourcePostgresRole) PrepareState(input *resources.PostgresRole) *PostgresRoleState {
	return &PostgresRoleState{
		RoleId:       input.RoleId,
		Parent:       input.Parent,
		RoleRoleSpec: input.RoleRoleSpec,
	}
}

func (*ResourcePostgresRole) RemapState(remote *postgres.Role) *PostgresRoleState {
	var roleID string
	if remote.Status != nil {
		roleID = remote.Status.RoleId
	}
	return &PostgresRoleState{
		RoleId: roleID,
		Parent: remote.Parent,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		// Use an empty struct (not nil) so field-level diffing works correctly.
		RoleRoleSpec: postgres.RoleRoleSpec{
			Attributes:      nil,
			AuthMethod:      "",
			IdentityType:    "",
			MembershipRoles: nil,
			PostgresRole:    "",
			ForceSendFields: nil,
		},
	}
}

func (r *ResourcePostgresRole) DoRead(ctx context.Context, id string) (*postgres.Role, error) {
	return r.client.Postgres.GetRole(ctx, postgres.GetRoleRequest{Name: id})
}

func (r *ResourcePostgresRole) DoCreate(ctx context.Context, config *PostgresRoleState) (string, *postgres.Role, error) {
	waiter, err := r.client.Postgres.CreateRole(ctx, postgres.CreateRoleRequest{
		RoleId: config.RoleId,
		Parent: config.Parent,
		Role: postgres.Role{
			Spec: &config.RoleRoleSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		ForceSendFields: nil,
	})
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	return result.Name, result, nil
}

func (r *ResourcePostgresRole) DoUpdate(ctx context.Context, id string, config *PostgresRoleState, entry *PlanEntry) (*postgres.Role, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// Prefix with "spec." because the API expects paths relative to the Role
	// object, not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.")

	waiter, err := r.client.Postgres.UpdateRole(ctx, postgres.UpdateRoleRequest{
		Name: id,
		Role: postgres.Role{
			Spec: &config.RoleRoleSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		UpdateMask: fieldmask.FieldMask{
			Paths: fieldPaths,
		},
	})
	if err != nil {
		return nil, err
	}

	return waiter.Wait(ctx)
}

func (r *ResourcePostgresRole) DoDelete(ctx context.Context, id string, _ *PostgresRoleState) error {
	waiter, err := r.client.Postgres.DeleteRole(ctx, postgres.DeleteRoleRequest{
		Name: id,

		// ReassignOwnedTo is intentionally unset; honoring it would require
		// user-facing config we don't expose, and it would spin up compute to
		// run reassignment SQL.
		ReassignOwnedTo: "",
		ForceSendFields: nil,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
