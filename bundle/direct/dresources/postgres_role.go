package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
)

type ResourcePostgresRole struct {
	client *sdkclients.Clients
}

// PostgresRoleState keeps role_id and parent as separate fields rather than a
// pre-joined hierarchical name. That alignment matters because bundle variable
// resolution only rewrites state fields whose JSON paths appear in the input
// config's refs map (parent, role_id, etc.); a synthesized "name" field built
// from input.Parent at PrepareState time would keep the literal ${...} string
// when parent comes from a resource reference.
type PostgresRoleState struct {
	postgres.Role_RoleSpec

	// RoleId is the leaf id, matching the user-facing config.
	RoleId string `json:"role_id"`

	// Parent is "projects/{project_id}/branches/{branch_id}".
	Parent string `json:"parent"`

	// ReplaceExisting takes over an existing role with the same ID on create
	// instead of returning ALREADY_EXISTS. Input-only: not returned by the GET API.
	ReplaceExisting bool `json:"replace_existing,omitempty"`
}

// PostgresRoleRemote is the return type for DoRead. It embeds Role_RoleSpec so that
// all paths in StateType are valid paths in RemoteType, enabling drift detection
// for spec fields once the backend echoes spec on GET.
type PostgresRoleRemote struct {
	postgres.Role_RoleSpec

	RoleId string `json:"role_id,omitempty"`
	Parent string `json:"parent,omitempty"`

	Name       string                    `json:"name,omitempty"`
	Status     *postgres.Role_RoleStatus `json:"status,omitempty"`
	CreateTime *types.Time               `json:"create_time,omitempty"`
	UpdateTime *types.Time               `json:"update_time,omitempty"`
}

func (*ResourcePostgresRole) New(client *sdkclients.Clients) *ResourcePostgresRole {
	return &ResourcePostgresRole{client: client}
}

func (*ResourcePostgresRole) PrepareState(input *resources.PostgresRole) *PostgresRoleState {
	return &PostgresRoleState{
		RoleId:          input.RoleId,
		Parent:          input.Parent,
		ReplaceExisting: input.ReplaceExisting,
		Role_RoleSpec:   input.Role_RoleSpec,
	}
}

func (*ResourcePostgresRole) RemapState(remote *PostgresRoleRemote) *PostgresRoleState {
	return &PostgresRoleState{
		RoleId: remote.RoleId,
		Parent: remote.Parent,

		// replace_existing is a create-time-only flag; the GET API never returns
		// it, so RemapState leaves it false.
		ReplaceExisting: false,

		Role_RoleSpec: remote.Role_RoleSpec,
	}
}

// makePostgresRoleRemote converts the SDK Role into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and postgres_roles.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresRoleRemote(resp *sdkpostgres.Role) (*PostgresRoleRemote, error) {
	role, err := postgres.RoleFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.Role_RoleSpec
	if role.Spec != nil {
		spec = *role.Spec
	}
	return &PostgresRoleRemote{
		Role_RoleSpec: spec,
		RoleId:        utils.Deref(role.RoleId),
		Parent:        utils.Deref(role.Parent),
		Name:          utils.Deref(role.Name),
		Status:        role.Status,
		CreateTime:    role.CreateTime,
		UpdateTime:    role.UpdateTime,
	}, nil
}

func (r *ResourcePostgresRole) DoRead(ctx context.Context, id string) (*PostgresRoleRemote, error) {
	role, err := r.client.Postgres.GetRole(ctx, sdkpostgres.GetRoleRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresRoleRemote(role)
}

func (r *ResourcePostgresRole) DoCreate(ctx context.Context, config *PostgresRoleState) (string, *PostgresRoleRemote, error) {
	request, err := postgres.CreateRoleRequestToSDK(&postgres.CreateRoleRequest{
		RoleId:          nonEmpty(config.RoleId),
		Parent:          nonEmpty(config.Parent),
		ReplaceExisting: trueOrNil(config.ReplaceExisting),
		Role:            &postgres.Role{Spec: &config.Role_RoleSpec},
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateRole(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote, err := makePostgresRoleRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func (r *ResourcePostgresRole) DoUpdate(ctx context.Context, id string, config *PostgresRoleState, entry *PlanEntry) (*PostgresRoleRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// Prefix with "spec." because the API expects paths relative to the Role
	// object, not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.", nil)

	request, err := postgres.UpdateRoleRequestToSDK(&postgres.UpdateRoleRequest{
		Role:       &postgres.Role{Name: new(id), Spec: &config.Role_RoleSpec},
		UpdateMask: new(strings.Join(fieldPaths, ",")),
	})
	if err != nil {
		return nil, err
	}
	waiter, err := r.client.Postgres.UpdateRole(ctx, *request)
	if err != nil {
		return nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresRoleRemote(result)
}

func (r *ResourcePostgresRole) DoDelete(ctx context.Context, id string, _ *PostgresRoleState) error {
	waiter, err := r.client.Postgres.DeleteRole(ctx, sdkpostgres.DeleteRoleRequest{Name: new(id)})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
