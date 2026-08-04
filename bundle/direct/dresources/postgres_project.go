package dresources

import (
	"context"
	"slices"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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
	CreateTime          *sdktime.Time                 `json:"create_time,omitempty"`
	DeleteTime          *sdktime.Time                 `json:"delete_time,omitempty"`
	PurgeTime           *sdktime.Time                 `json:"purge_time,omitempty"`
	UpdateTime          *sdktime.Time                 `json:"update_time,omitempty"`
}

// Custom marshaler needed because embedded ProjectSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *PostgresProjectRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresProjectRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourcePostgresProject struct {
	client *databricks.WorkspaceClient
}

type PostgresProjectState = resources.PostgresProjectConfig

func (*ResourcePostgresProject) New(client *databricks.WorkspaceClient) *ResourcePostgresProject {
	return &ResourcePostgresProject{client: client}
}

func (*ResourcePostgresProject) PrepareState(input *resources.PostgresProject) *PostgresProjectState {
	return &PostgresProjectState{
		ProjectId:       input.ProjectId,
		PurgeOnDelete:   input.PurgeOnDelete,
		ProjectSpec:     input.ProjectSpec,
		ForceSendFields: input.ForceSendFields,
	}
}

func (*ResourcePostgresProject) RemapState(remote *PostgresProjectRemote) *PostgresProjectState {
	return &PostgresProjectState{
		ProjectId:   remote.ProjectId,
		ProjectSpec: remote.ProjectSpec,

		// purge_on_delete is a delete-time query parameter; the GET API never
		// returns it, so RemapState leaves it false.
		PurgeOnDelete:   false,
		ForceSendFields: nil,
	}
}

// makePostgresProjectRemote converts the SDK Project into the embedded remote shape.
// GET does not echo spec today (only status is returned); the embedded spec fields
// stay at their zero values, and resources.yml suppresses phantom drift via
// ignore_remote_changes with reason spec:input_only.
func makePostgresProjectRemote(project *postgres.Project) *PostgresProjectRemote {
	var spec postgres.ProjectSpec
	if project.Spec != nil {
		spec = *project.Spec
	}
	var projectID string
	if project.Status != nil {
		projectID = project.Status.ProjectId
	}
	return &PostgresProjectRemote{
		ProjectSpec:         spec,
		ProjectId:           projectID,
		InitialEndpointSpec: project.InitialEndpointSpec,
		Name:                project.Name,
		Status:              project.Status,
		Uid:                 project.Uid,
		CreateTime:          project.CreateTime,
		DeleteTime:          project.DeleteTime,
		PurgeTime:           project.PurgeTime,
		UpdateTime:          project.UpdateTime,
	}
}

func (r *ResourcePostgresProject) DoRead(ctx context.Context, id string) (*PostgresProjectRemote, error) {
	project, err := r.client.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresProjectRemote(project), nil
}

func (r *ResourcePostgresProject) DoCreate(ctx context.Context, config *PostgresProjectState) (string, *PostgresProjectRemote, error) {
	waiter, err := r.client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
		ProjectId: config.ProjectId,
		Project: postgres.Project{
			Spec:                &config.ProjectSpec,
			InitialEndpointSpec: nil,

			// Output-only fields.
			CreateTime:      nil,
			DeleteTime:      nil,
			Name:            "",
			PurgeTime:       nil,
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
	})
	if err != nil {
		return "", nil, err
	}

	// Wait for the project to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote := makePostgresProjectRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresProject) DoUpdate(ctx context.Context, id string, config *PostgresProjectState, entry *PlanEntry) (*PostgresProjectRemote, error) {
	// Build the mask from the plan's change list and prefix with "spec." (the
	// API expects paths relative to Project). The API rejects mask entries
	// that aren't also populated in the request body, and a wildcard "*"
	// expands to nested attributes the body would have to set too — so we
	// can't use a static all-fields mask. The change list naturally tracks
	// what the user actually set, so the body and mask stay consistent.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.")

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

	waiter, err := r.client.Postgres.UpdateProject(ctx, postgres.UpdateProjectRequest{
		Project: postgres.Project{
			Spec:                &config.ProjectSpec,
			InitialEndpointSpec: nil,

			// Output-only fields.
			CreateTime:      nil,
			DeleteTime:      nil,
			Name:            "",
			PurgeTime:       nil,
			Status:          nil,
			Uid:             "",
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		Name: id,
		UpdateMask: fieldmask.FieldMask{
			Paths: fieldPaths,
		},
	})
	if err != nil {
		return nil, err
	}

	// Wait for the update to complete
	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresProjectRemote(result), nil
}

func (r *ResourcePostgresProject) DoDelete(ctx context.Context, id string, state *PostgresProjectState) error {
	waiter, err := r.client.Postgres.DeleteProject(ctx, postgres.DeleteProjectRequest{
		Name:            id,
		Purge:           state.PurgeOnDelete,
		ForceSendFields: nil,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
