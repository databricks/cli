package dresources

import (
	"context"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/core/types"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
)

// PostgresProjectRemote is the return type for DoRead. It embeds ProjectSpec so
// that all paths in StateType are valid paths in RemoteType, enabling drift
// detection for spec fields once the backend echoes spec on GET.
type PostgresProjectRemote struct {
	postgres.ProjectSpec

	ProjectId string `json:"project_id,omitempty"`

	InitialEndpointSpec *postgres.InitialEndpointSpec `json:"initial_endpoint_spec,omitempty"`
	Name                string                        `json:"name,omitempty"`
	Status              *postgres.ProjectStatus       `json:"status,omitempty"`
	Uid                 string                        `json:"uid,omitempty"`
	CreateTime          *types.Time                   `json:"create_time,omitempty"`
	DeleteTime          *types.Time                   `json:"delete_time,omitempty"`
	PurgeTime           *types.Time                   `json:"purge_time,omitempty"`
	UpdateTime          *types.Time                   `json:"update_time,omitempty"`
}

type ResourcePostgresProject struct {
	client *sdkclients.Clients
}

type PostgresProjectState = resources.PostgresProjectConfig

func (*ResourcePostgresProject) New(client *sdkclients.Clients) *ResourcePostgresProject {
	return &ResourcePostgresProject{client: client}
}

func (*ResourcePostgresProject) PrepareState(input *resources.PostgresProject) *PostgresProjectState {
	return &PostgresProjectState{
		ProjectId:     input.ProjectId,
		PurgeOnDelete: input.PurgeOnDelete,
		ProjectSpec:   input.ProjectSpec,
	}
}

func (*ResourcePostgresProject) RemapState(remote *PostgresProjectRemote) *PostgresProjectState {
	return &PostgresProjectState{
		ProjectId:   remote.ProjectId,
		ProjectSpec: remote.ProjectSpec,

		// purge_on_delete is a delete-time query parameter; the GET API never
		// returns it, so RemapState leaves it false.
		PurgeOnDelete: false,
	}
}

// makePostgresProjectRemote converts the SDK Project into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and postgres_projects.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresProjectRemote(resp *sdkpostgres.Project) (*PostgresProjectRemote, error) {
	project, err := postgres.ProjectFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.ProjectSpec
	if project.Spec != nil {
		spec = *project.Spec
	}
	return &PostgresProjectRemote{
		ProjectSpec:         spec,
		ProjectId:           utils.Deref(project.ProjectId),
		InitialEndpointSpec: project.InitialEndpointSpec,
		Name:                utils.Deref(project.Name),
		Status:              project.Status,
		Uid:                 utils.Deref(project.Uid),
		CreateTime:          project.CreateTime,
		DeleteTime:          project.DeleteTime,
		PurgeTime:           project.PurgeTime,
		UpdateTime:          project.UpdateTime,
	}, nil
}

func (r *ResourcePostgresProject) DoRead(ctx context.Context, id string) (*PostgresProjectRemote, error) {
	project, err := r.client.Postgres.GetProject(ctx, sdkpostgres.GetProjectRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresProjectRemote(project)
}

func (r *ResourcePostgresProject) DoCreate(ctx context.Context, config *PostgresProjectState) (string, *PostgresProjectRemote, error) {
	// Only spec is set; the other Project fields are output-only.
	request, err := postgres.CreateProjectRequestToSDK(&postgres.CreateProjectRequest{
		ProjectId: new(config.ProjectId),
		Project:   &postgres.Project{Spec: &config.ProjectSpec},
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateProject(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	// Wait for the project to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote, err := makePostgresProjectRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

// The default endpoint settings carry the same suspension oneof as an endpoint does;
// see branchOneofGroups.
var projectOneofGroups = map[string]string{
	"default_endpoint_settings.no_suspension":            "default_endpoint_settings.suspension",
	"default_endpoint_settings.suspend_timeout_duration": "default_endpoint_settings.suspension",
}

func (r *ResourcePostgresProject) DoUpdate(ctx context.Context, id string, config *PostgresProjectState, entry *PlanEntry) (*PostgresProjectRemote, error) {
	// Build the mask from the plan's change list and prefix with "spec." (the
	// API expects paths relative to Project). The API rejects mask entries
	// that aren't also populated in the request body, and a wildcard "*"
	// expands to nested attributes the body would have to set too — so we
	// can't use a static all-fields mask. The change list naturally tracks
	// what the user actually set, so the body and mask stay consistent.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.", projectOneofGroups)

	// purge_on_delete is an input-only flag consulted at delete time; it is
	// not a spec field. Strip it from the mask so toggling it between deploys
	// becomes a state-only refresh (the framework saves newState when this
	// returns nil error).
	fieldPaths = slices.DeleteFunc(fieldPaths, func(p string) bool {
		return p == "spec.purge_on_delete"
	})
	if len(fieldPaths) == 0 {
		return nil, nil
	}

	// Only name (which addresses the project) and spec are set; the other
	// Project fields are output-only.
	request, err := postgres.UpdateProjectRequestToSDK(&postgres.UpdateProjectRequest{
		Project: &postgres.Project{
			Name: new(id),
			Spec: &config.ProjectSpec,
		},
		UpdateMask: new(strings.Join(fieldPaths, ",")),
	})
	if err != nil {
		return nil, err
	}
	waiter, err := r.client.Postgres.UpdateProject(ctx, *request)
	if err != nil {
		return nil, err
	}

	// Wait for the update to complete
	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresProjectRemote(result)
}

func (r *ResourcePostgresProject) DoDelete(ctx context.Context, id string, state *PostgresProjectState) error {
	waiter, err := r.client.Postgres.DeleteProject(ctx, sdkpostgres.DeleteProjectRequest{
		Name:  new(id),
		Purge: trueOrNil(state.PurgeOnDelete),
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
