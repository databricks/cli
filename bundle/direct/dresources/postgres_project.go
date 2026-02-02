package dresources

import (
	"context"
	"errors"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresProject struct {
	client *databricks.WorkspaceClient
}

// PostgresProjectState contains only the fields needed for creation/update.
// It does NOT include output-only fields like Name, which are only available after API response.
type PostgresProjectState struct {
	ProjectId string                `json:"project_id,omitempty"`
	Spec      *postgres.ProjectSpec `json:"spec,omitempty"`
}

func (*ResourcePostgresProject) New(client *databricks.WorkspaceClient) *ResourcePostgresProject {
	return &ResourcePostgresProject{client: client}
}

func (*ResourcePostgresProject) PrepareState(input *resources.PostgresProject) *PostgresProjectState {
	return &PostgresProjectState{
		Spec:      &input.ProjectSpec,
		ProjectId: input.ProjectId,
	}
}

func (*ResourcePostgresProject) RemapState(remote *postgres.Project) *PostgresProjectState {
	// Extract project_id from hierarchical name: "projects/{project_id}"
	projectId := strings.TrimPrefix(remote.Name, "projects/")

	return &PostgresProjectState{
		ProjectId: projectId,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		Spec: nil,
	}
}

func (r *ResourcePostgresProject) DoRead(ctx context.Context, id string) (*postgres.Project, error) {
	return r.client.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: id})
}

func (r *ResourcePostgresProject) DoCreate(ctx context.Context, config *PostgresProjectState) (string, *postgres.Project, error) {
	projectId := config.ProjectId
	if projectId == "" {
		return "", nil, errors.New("project_id must be specified")
	}

	waiter, err := r.client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
		ProjectId: projectId,
		Project: postgres.Project{
			Spec: config.Spec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
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

	return result.Name, result, nil
}

func (r *ResourcePostgresProject) DoUpdate(ctx context.Context, id string, config *PostgresProjectState, changes Changes) (*postgres.Project, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	fieldPaths := collectUpdatePaths(changes)

	waiter, err := r.client.Postgres.UpdateProject(ctx, postgres.UpdateProjectRequest{
		Project: postgres.Project{
			Spec: config.Spec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
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
	return result, err
}

func (r *ResourcePostgresProject) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Postgres.DeleteProject(ctx, postgres.DeleteProjectRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
