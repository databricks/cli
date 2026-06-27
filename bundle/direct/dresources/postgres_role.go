package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
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

// PostgresRoleRemote is the return type for DoRead. It embeds RoleRoleSpec so that
// all paths in StateType are valid paths in RemoteType, enabling drift detection
// for spec fields once the backend echoes spec on GET.
type PostgresRoleRemote struct {
	postgres.RoleRoleSpec

	RoleId string `json:"role_id,omitempty"`
	Parent string `json:"parent,omitempty"`

	Name       string                   `json:"name,omitempty"`
	Status     *postgres.RoleRoleStatus `json:"status,omitempty"`
	CreateTime *sdktime.Time            `json:"create_time,omitempty"`
	UpdateTime *sdktime.Time            `json:"update_time,omitempty"`
}

// Custom marshalers needed because embedded RoleRoleSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *PostgresRoleState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresRoleState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (s *PostgresRoleRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresRoleRemote) MarshalJSON() ([]byte, error) {
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

func (*ResourcePostgresRole) RemapState(remote *PostgresRoleRemote) *PostgresRoleState {
	return &PostgresRoleState{
		RoleId:       remote.RoleId,
		Parent:       remote.Parent,
		RoleRoleSpec: remote.RoleRoleSpec,
	}
}

// makePostgresRoleRemote converts the SDK Role into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and resources.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresRoleRemote(role *postgres.Role) *PostgresRoleRemote {
	var spec postgres.RoleRoleSpec
	if role.Spec != nil {
		spec = *role.Spec
	}
	var roleID string
	if role.Status != nil {
		roleID = role.Status.RoleId
	}
	return &PostgresRoleRemote{
		RoleRoleSpec: spec,
		RoleId:       roleID,
		Parent:       role.Parent,
		Name:         role.Name,
		Status:       role.Status,
		CreateTime:   role.CreateTime,
		UpdateTime:   role.UpdateTime,
	}
}

func (r *ResourcePostgresRole) DoRead(ctx context.Context, id string) (*PostgresRoleRemote, error) {
	role, err := r.client.Postgres.GetRole(ctx, postgres.GetRoleRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresRoleRemote(role), nil
}

func (r *ResourcePostgresRole) DoCreate(ctx context.Context, _ *StateSaver, config *PostgresRoleState) (string, *PostgresRoleRemote, error) {
	waiter, err := r.client.Postgres.CreateRole(ctx, postgres.CreateRoleRequest{
		RoleId: config.RoleId,
		Parent: config.Parent,
		Role: postgres.Role{
			Spec: &config.RoleRoleSpec,

			// Output-only fields.
			RoleId:          "",
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

	remote := makePostgresRoleRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresRole) DoUpdate(ctx context.Context, _ *StateSaver, id string, config *PostgresRoleState, entry *PlanEntry) (*PostgresRoleRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// Prefix with "spec." because the API expects paths relative to the Role
	// object, not relative to our flattened state type.
	fieldPaths := collectLeafUpdatePathsWithPrefix(entry.Changes, "spec.")

	waiter, err := r.client.Postgres.UpdateRole(ctx, postgres.UpdateRoleRequest{
		Name: id,
		Role: postgres.Role{
			Spec: &config.RoleRoleSpec,

			// Output-only fields.
			RoleId:          "",
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

	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresRoleRemote(result), nil
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
