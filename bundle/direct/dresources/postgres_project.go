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
		ProjectId:   input.ProjectId,
		ProjectSpec: input.ProjectSpec,
	}
}

func (*ResourcePostgresProject) RemapState(remote *PostgresProjectRemote) *PostgresProjectState {
	return &PostgresProjectState{
		ProjectId:   remote.ProjectId,
		ProjectSpec: remote.ProjectSpec,
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

func (r *ResourcePostgresProject) DoCreate(ctx context.Context, _ *Engine, config *PostgresProjectState) (string, *PostgresProjectRemote, error) {
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

func (r *ResourcePostgresProject) DoUpdate(ctx context.Context, _ *Engine, id string, config *PostgresProjectState, entry *PlanEntry) (*PostgresProjectRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Project object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.")

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

func (r *ResourcePostgresProject) DoDelete(ctx context.Context, id string, _ *PostgresProjectState) error {
	waiter, err := r.client.Postgres.DeleteProject(ctx, postgres.DeleteProjectRequest{
		Name:            id,
		Purge:           false,
		ForceSendFields: nil,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
