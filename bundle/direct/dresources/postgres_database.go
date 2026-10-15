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

// PostgresDatabaseRemote is the return type for DoRead. It embeds
// Database_DatabaseSpec so that all paths in StateType are valid paths in
// RemoteType, enabling drift detection for spec fields once the backend echoes
// spec on GET.
type PostgresDatabaseRemote struct {
	postgres.Database_DatabaseSpec

	DatabaseId string `json:"database_id,omitempty"`
	Parent     string `json:"parent,omitempty"`

	Name       string                            `json:"name,omitempty"`
	Status     *postgres.Database_DatabaseStatus `json:"status,omitempty"`
	CreateTime *types.Time                       `json:"create_time,omitempty"`
	UpdateTime *types.Time                       `json:"update_time,omitempty"`
}

type ResourcePostgresDatabase struct {
	client *sdkclients.Clients
}

type PostgresDatabaseState = resources.PostgresDatabaseConfig

func (*ResourcePostgresDatabase) New(client *sdkclients.Clients) *ResourcePostgresDatabase {
	return &ResourcePostgresDatabase{client: client}
}

func (*ResourcePostgresDatabase) PrepareState(input *resources.PostgresDatabase) *PostgresDatabaseState {
	return &PostgresDatabaseState{
		DatabaseId:            input.DatabaseId,
		Parent:                input.Parent,
		ReplaceExisting:       input.ReplaceExisting,
		Database_DatabaseSpec: input.Database_DatabaseSpec,
	}
}

func (*ResourcePostgresDatabase) RemapState(remote *PostgresDatabaseRemote) *PostgresDatabaseState {
	return &PostgresDatabaseState{
		DatabaseId: remote.DatabaseId,
		Parent:     remote.Parent,

		// replace_existing is a create-time-only flag; the GET API never returns
		// it, so RemapState leaves it false.
		ReplaceExisting: false,

		Database_DatabaseSpec: remote.Database_DatabaseSpec,
	}
}

// makePostgresDatabaseRemote converts the SDK Database into the embedded remote
// shape. GET does not echo spec today (only status is returned); the embedded
// spec fields stay at their zero values, and postgres_databases.yml suppresses phantom
// drift via ignore_remote_changes with reason spec:input_only.
func makePostgresDatabaseRemote(resp *sdkpostgres.Database) (*PostgresDatabaseRemote, error) {
	database, err := postgres.DatabaseFromSDK(resp)
	if err != nil {
		return nil, err
	}
	var spec postgres.Database_DatabaseSpec
	if database.Spec != nil {
		spec = *database.Spec
	}
	return &PostgresDatabaseRemote{
		Database_DatabaseSpec: spec,
		DatabaseId:            utils.Deref(database.DatabaseId),
		Parent:                utils.Deref(database.Parent),
		Name:                  utils.Deref(database.Name),
		Status:                database.Status,
		CreateTime:            database.CreateTime,
		UpdateTime:            database.UpdateTime,
	}, nil
}

func (r *ResourcePostgresDatabase) DoRead(ctx context.Context, id string) (*PostgresDatabaseRemote, error) {
	database, err := r.client.Postgres.GetDatabase(ctx, sdkpostgres.GetDatabaseRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresDatabaseRemote(database)
}

func (r *ResourcePostgresDatabase) DoCreate(ctx context.Context, config *PostgresDatabaseState) (string, *PostgresDatabaseRemote, error) {
	request, err := postgres.CreateDatabaseRequestToSDK(&postgres.CreateDatabaseRequest{
		DatabaseId:      nonEmpty(config.DatabaseId),
		Parent:          nonEmpty(config.Parent),
		ReplaceExisting: trueOrNil(config.ReplaceExisting),
		Database:        &postgres.Database{Spec: &config.Database_DatabaseSpec},
	})
	if err != nil {
		return "", nil, err
	}
	waiter, err := r.client.Postgres.CreateDatabase(ctx, *request)
	if err != nil {
		return "", nil, err
	}

	// Wait for the database to be ready (long-running operation)
	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote, err := makePostgresDatabaseRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func (r *ResourcePostgresDatabase) DoUpdate(ctx context.Context, id string, config *PostgresDatabaseState, entry *PlanEntry) (*PostgresDatabaseRemote, error) {
	// Build update mask from fields that have action="update" in the changes map.
	// This excludes immutable fields and fields that haven't changed.
	// Prefix with "spec." because the API expects paths relative to the Database object,
	// not relative to our flattened state type.
	fieldPaths := collectUpdatePathsWithPrefix(entry.Changes, "spec.", nil)

	request, err := postgres.UpdateDatabaseRequestToSDK(&postgres.UpdateDatabaseRequest{
		Database:   &postgres.Database{Name: new(id), Spec: &config.Database_DatabaseSpec},
		UpdateMask: new(strings.Join(fieldPaths, ",")),
	})
	if err != nil {
		return nil, err
	}
	waiter, err := r.client.Postgres.UpdateDatabase(ctx, *request)
	if err != nil {
		return nil, err
	}

	// Wait for the update to complete
	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresDatabaseRemote(result)
}

func (r *ResourcePostgresDatabase) DoDelete(ctx context.Context, id string, _ *PostgresDatabaseState) error {
	waiter, err := r.client.Postgres.DeleteDatabase(ctx, sdkpostgres.DeleteDatabaseRequest{Name: new(id)})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
