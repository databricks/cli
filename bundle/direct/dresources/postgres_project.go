package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

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

func (*ResourcePostgresProject) RemapState(remote *postgres.Project) *PostgresProjectState {
	// Extract project_id from hierarchical name: "projects/{project_id}"
	// TODO: log error when we have access to the context
	components, _ := ParsePostgresName(remote.Name)

	return &PostgresProjectState{
		ProjectId: components.ProjectID,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		// Use an empty struct (not nil) so field-level diffing works correctly.
		ProjectSpec: postgres.ProjectSpec{
			BudgetPolicyId:           "",
			CustomTags:               nil,
			DefaultEndpointSettings:  nil,
			DisplayName:              "",
			HistoryRetentionDuration: nil,
			PgVersion:                0,
			ForceSendFields:          nil,
		},
	}
}

func (r *ResourcePostgresProject) DoRead(ctx context.Context, id string) (*postgres.Project, error) {
	return r.client.Postgres.GetProject(ctx, postgres.GetProjectRequest{Name: id})
}

func (r *ResourcePostgresProject) DoCreate(ctx context.Context, config *PostgresProjectState) (string, *postgres.Project, error) {
	waiter, err := r.client.Postgres.CreateProject(ctx, postgres.CreateProjectRequest{
		ProjectId: config.ProjectId,
		Project: postgres.Project{
			Spec:                &config.ProjectSpec,
			InitialEndpointSpec: nil,

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
	// Prefix with "spec." because the API expects paths relative to the Project object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(changes, "spec.")

	waiter, err := r.client.Postgres.UpdateProject(ctx, postgres.UpdateProjectRequest{
		Project: postgres.Project{
			Spec:                &config.ProjectSpec,
			InitialEndpointSpec: nil,

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
