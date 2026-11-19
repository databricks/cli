package dresources

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
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

	// ReplaceExisting takes over an existing role with the same ID on create
	// instead of returning ALREADY_EXISTS. Input-only: not returned by the GET API.
	ReplaceExisting *bool `json:"replace_existing,omitempty"`
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
		RoleId:          input.RoleId,
		Parent:          input.Parent,
		ReplaceExisting: input.ReplaceExisting,
		RoleRoleSpec:    input.RoleRoleSpec,
	}
}

func (*ResourcePostgresRole) RemapState(remote *PostgresRoleRemote) *PostgresRoleState {
	return &PostgresRoleState{
		RoleId: remote.RoleId,
		Parent: remote.Parent,

		// replace_existing is a create-time-only flag; the GET API never returns
		// it, so RemapState leaves it unset.
		ReplaceExisting: nil,

		RoleRoleSpec: remote.RoleRoleSpec,
	}
}

// makePostgresRoleRemote converts the SDK Role into the embedded remote shape.
// Since 2026-11 GET echoes spec. Fields still marked spec:input_only in
// postgres_roles.generated.yml keep remote changes ignored until the OpenAPI
// annotations, and the config generated from them, catch up.
func makePostgresRoleRemote(role *postgres.Role) *PostgresRoleRemote {
	var spec postgres.RoleRoleSpec
	if role.Spec != nil {
		spec = *role.Spec
	}
	return &PostgresRoleRemote{
		RoleRoleSpec: spec,
		RoleId:       role.RoleId,
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

func (r *ResourcePostgresRole) DoCreate(ctx context.Context, config *PostgresRoleState) (string, *PostgresRoleRemote, error) {
	req := postgres.CreateRoleRequest{
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
		ReplaceExisting: config.ReplaceExisting != nil && *config.ReplaceExisting,
		ForceSendFields: nil,
	}
	start := time.Now()
	waiter, err := r.client.Postgres.CreateRole(ctx, req)
	if err != nil {
		return r.recoverFailedCreate(ctx, req, start, err)
	}
	return r.waitCreated(ctx, waiter)
}

// waitCreated waits for the create operation of a role that a create request returned.
func (r *ResourcePostgresRole) waitCreated(ctx context.Context, waiter postgres.CreateRoleOperationInterface) (string, *PostgresRoleRemote, error) {
	result, err := waitCreateOperation(ctx, waiter.Wait)
	if err != nil {
		return "", nil, err
	}
	remote := makePostgresRoleRemote(result)
	return remote.Name, remote, nil
}

// recoverFailedCreate handles a create request that failed; see recoverPostgresCreate. Role
// creates on a just-created branch are answered with 500 INTERNAL_ERROR now and then, and the
// request may have created the role anyway, so look for it before repeating the request.
func (r *ResourcePostgresRole) recoverFailedCreate(ctx context.Context, req postgres.CreateRoleRequest, start time.Time, createErr error) (string, *PostgresRoleRemote, error) {
	if ctx.Err() != nil || !retry.IsTransient(ctx, createErr) || req.RoleId == "" {
		return "", nil, createErr
	}
	// A role found after a replace may predate it, and repeating a replace is safe.
	if req.ReplaceExisting {
		return "", nil, retrySafe(createErr)
	}

	name := req.Parent + "/roles/" + req.RoleId
	return recoverPostgresCreate(ctx, postgresCreateRecovery[postgres.Role, *PostgresRoleRemote]{
		name: name,
		get: func(ctx context.Context) (*postgres.Role, error) {
			return r.client.Postgres.GetRole(ctx, postgres.GetRoleRequest{Name: name})
		},
		create: func(ctx context.Context) (func() (string, *PostgresRoleRemote, error), error) {
			waiter, err := r.client.Postgres.CreateRole(ctx, req)
			if err != nil {
				return nil, err
			}
			return func() (string, *PostgresRoleRemote, error) { return r.waitCreated(ctx, waiter) }, nil
		},
		use: func(ctx context.Context, role *postgres.Role) (string, *PostgresRoleRemote, error) {
			if !createdByRequestAt(role.CreateTime, start) {
				return "", nil, fmt.Errorf("%w; role %s exists but was not created by this deployment", createErr, role.Name)
			}
			// Info, not Warn: the deploy got the role it asked for, so there is nothing to act on.
			log.Infof(ctx, "using role %s, created despite the failed create request: %s", role.Name, createErr)
			remote := makePostgresRoleRemote(role)
			return remote.Name, remote, nil
		},
	}, createErr)
}

func (r *ResourcePostgresRole) DoUpdate(ctx context.Context, id string, config *PostgresRoleState, _ *PlanEntry) (*PostgresRoleRemote, error) {
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
			Paths: postgresSpecUpdateMask,
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
