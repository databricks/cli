package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

type ResourcePostgresDatabase struct {
	client *databricks.WorkspaceClient
}

type PostgresDatabaseState = resources.PostgresDatabaseConfig

func (*ResourcePostgresDatabase) New(client *databricks.WorkspaceClient) *ResourcePostgresDatabase {
	return &ResourcePostgresDatabase{client: client}
}

func (*ResourcePostgresDatabase) PrepareState(input *resources.PostgresDatabase) *PostgresDatabaseState {
	return &PostgresDatabaseState{
		DatabaseId:           input.DatabaseId,
		Parent:               input.Parent,
		DatabaseDatabaseSpec: input.DatabaseDatabaseSpec,
	}
}

func (*ResourcePostgresDatabase) RemapState(remote *postgres.Database) *PostgresDatabaseState {
	// Extract database_id from hierarchical name: "projects/{project_id}/branches/{branch_id}/databases/{database_id}"
	// TODO: log error when we have access to the context
	components, _ := ParsePostgresName(remote.Name)

	return &PostgresDatabaseState{
		DatabaseId: components.DatabaseID,
		Parent:     remote.Parent,

		// The read API does not return the spec, only the status.
		// This means we cannot detect remote drift for spec fields.
		// Use an empty struct (not nil) so field-level diffing works correctly.
		DatabaseDatabaseSpec: postgres.DatabaseDatabaseSpec{
			PostgresDatabase: "",
			Role:             "",
			ForceSendFields:  nil,
		},
	}
}

func (r *ResourcePostgresDatabase) DoRead(ctx context.Context, id string) (*postgres.Database, error) {
	return r.client.Postgres.GetDatabase(ctx, postgres.GetDatabaseRequest{Name: id})
}

func (r *ResourcePostgresDatabase) DoCreate(ctx context.Context, config *PostgresDatabaseState) (string, *postgres.Database, error) {
	waiter, err := r.client.Postgres.CreateDatabase(ctx, postgres.CreateDatabaseRequest{
		DatabaseId: config.DatabaseId,
		Parent:     config.Parent,
		Database: postgres.Database{
			Spec: &config.DatabaseDatabaseSpec,

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

	// Wait for the database to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	return result.Name, result, nil
}

func (r *ResourcePostgresDatabase) DoUpdate(ctx context.Context, id string, config *PostgresDatabaseState, entry *PlanEntry) (*postgres.Database, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Database object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.")

	waiter, err := r.client.Postgres.UpdateDatabase(ctx, postgres.UpdateDatabaseRequest{
		Database: postgres.Database{
			Spec: &config.DatabaseDatabaseSpec,

			// Output-only fields.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
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

func (r *ResourcePostgresDatabase) DoDelete(ctx context.Context, id string) error {
	waiter, err := r.client.Postgres.DeleteDatabase(ctx, postgres.DeleteDatabaseRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
