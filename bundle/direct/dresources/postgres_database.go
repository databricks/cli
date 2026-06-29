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

// PostgresDatabaseRemote is the return type for DoRead. It embeds
// DatabaseDatabaseSpec so that all paths in StateType are valid paths in
// RemoteType, enabling drift detection for spec fields once the backend echoes
// spec on GET.
type PostgresDatabaseRemote struct {
	postgres.DatabaseDatabaseSpec

	DatabaseId string `json:"database_id,omitempty"`
	Parent     string `json:"parent,omitempty"`

	Name       string                           `json:"name,omitempty"`
	Status     *postgres.DatabaseDatabaseStatus `json:"status,omitempty"`
	CreateTime *sdktime.Time                    `json:"create_time,omitempty"`
	UpdateTime *sdktime.Time                    `json:"update_time,omitempty"`
}

// Custom marshaler needed because embedded DatabaseDatabaseSpec has its own
// MarshalJSON which would otherwise take over and ignore the additional fields.
func (s *PostgresDatabaseRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresDatabaseRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

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

func (*ResourcePostgresDatabase) RemapState(remote *PostgresDatabaseRemote) *PostgresDatabaseState {
	return &PostgresDatabaseState{
		DatabaseId:           remote.DatabaseId,
		Parent:               remote.Parent,
		DatabaseDatabaseSpec: remote.DatabaseDatabaseSpec,
	}
}

// makePostgresDatabaseRemote converts the SDK Database into the embedded remote
// shape. GET does not echo spec today (only status is returned); the embedded
// spec fields stay at their zero values, and resources.yml suppresses phantom
// drift via ignore_remote_changes with reason spec:input_only.
func makePostgresDatabaseRemote(database *postgres.Database) *PostgresDatabaseRemote {
	var spec postgres.DatabaseDatabaseSpec
	if database.Spec != nil {
		spec = *database.Spec
	}
	return &PostgresDatabaseRemote{
		DatabaseDatabaseSpec: spec,
		DatabaseId:           database.DatabaseId,
		Parent:               database.Parent,
		Name:                 database.Name,
		Status:               database.Status,
		CreateTime:           database.CreateTime,
		UpdateTime:           database.UpdateTime,
	}
}

func (r *ResourcePostgresDatabase) DoRead(ctx context.Context, id string) (*PostgresDatabaseRemote, error) {
	database, err := r.client.Postgres.GetDatabase(ctx, postgres.GetDatabaseRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresDatabaseRemote(database), nil
}

func (r *ResourcePostgresDatabase) DoCreate(ctx context.Context, config *PostgresDatabaseState) (string, *PostgresDatabaseRemote, error) {
	waiter, err := r.client.Postgres.CreateDatabase(ctx, postgres.CreateDatabaseRequest{
		DatabaseId: config.DatabaseId,
		Parent:     config.Parent,
		Database: postgres.Database{
			Spec: &config.DatabaseDatabaseSpec,

			// Output-only fields.
			DatabaseId:      "",
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

	remote := makePostgresDatabaseRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresDatabase) DoUpdate(ctx context.Context, id string, config *PostgresDatabaseState, entry *PlanEntry) (*PostgresDatabaseRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Database object,
	// not relative to our flattened state type.
	fieldPaths := collectLeafUpdatePathsWithPrefix(entry.Changes, "spec.")

	waiter, err := r.client.Postgres.UpdateDatabase(ctx, postgres.UpdateDatabaseRequest{
		Database: postgres.Database{
			Spec: &config.DatabaseDatabaseSpec,

			// Output-only fields.
			DatabaseId:      "",
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
	if err != nil {
		return nil, err
	}
	return makePostgresDatabaseRemote(result), nil
}

func (r *ResourcePostgresDatabase) DoDelete(ctx context.Context, id string, _ *PostgresDatabaseState) error {
	waiter, err := r.client.Postgres.DeleteDatabase(ctx, postgres.DeleteDatabaseRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
