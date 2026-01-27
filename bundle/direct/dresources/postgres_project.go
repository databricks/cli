package dresources

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresProject struct {
	client *databricks.WorkspaceClient
}

// PostgresProjectState wraps the postgres.Project with additional fields needed for creation.
type PostgresProjectState struct {
	postgres.Project
	ProjectId string `json:"project_id,omitempty"`
}

func (*ResourcePostgresProject) New(client *databricks.WorkspaceClient) *ResourcePostgresProject {
	return &ResourcePostgresProject{client: client}
}

func (*ResourcePostgresProject) PrepareState(input *resources.PostgresProject) *PostgresProjectState {
	return &PostgresProjectState{
		Project: postgres.Project{
			Name: input.Name,
			Spec: &input.ProjectSpec,
		},
		ProjectId: input.ProjectId,
	}
}

func (*ResourcePostgresProject) RemapState(remote *postgres.Project) *PostgresProjectState {
	return &PostgresProjectState{
		Project: *remote,
		// ProjectId is not available in remote state, it's already part of the Name
	}
}

func (r *ResourcePostgresProject) DoRead(ctx context.Context, id string) (*postgres.Project, error) {
	return r.client.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: id})
}

func (r *ResourcePostgresProject) DoCreate(ctx context.Context, config *PostgresProjectState) (string, *postgres.Project, error) {
	projectId := config.ProjectId
	if projectId == "" {
		return "", nil, fmt.Errorf("project_id must be specified")
	}

	waiter, err := r.client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
		ProjectId: projectId,
		Project:   config.Project,
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

func (r *ResourcePostgresProject) DoUpdate(ctx context.Context, id string, config *PostgresProjectState, _ Changes) (*postgres.Project, error) {
	waiter, err := r.client.Postgres.UpdateProject(ctx, postgres.UpdateProjectRequest{
		Project: config.Project,
		Name:    id,
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"*"},
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
