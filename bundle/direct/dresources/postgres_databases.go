package dresources

import (
	"context"
	"time"

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
		ReplaceExisting:      input.ReplaceExisting,
		DatabaseDatabaseSpec: input.DatabaseDatabaseSpec,
	}
}

func (*ResourcePostgresDatabase) RemapState(remote *PostgresDatabaseRemote) *PostgresDatabaseState {
	return &PostgresDatabaseState{
		DatabaseId: remote.DatabaseId,
		Parent:     remote.Parent,

		// replace_existing is a create-time-only flag; the GET API never returns
		// it, so RemapState leaves it unset.
		ReplaceExisting: nil,

		DatabaseDatabaseSpec: remote.DatabaseDatabaseSpec,
	}
}

// makePostgresDatabaseRemote converts the SDK Database into the embedded remote
// shape. Since 2026-11 GET echoes spec. Fields still marked spec:input_only in
// postgres_databases.generated.yml keep remote changes ignored until the OpenAPI
// annotations, and the config generated from them, catch up.
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
	req := postgres.CreateDatabaseRequest{
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
		ReplaceExisting: config.ReplaceExisting != nil && *config.ReplaceExisting,
		ForceSendFields: nil,
	}
	start := time.Now()
	waiter, err := r.client.Postgres.CreateDatabase(ctx, req)
	if err != nil {
		return r.recoverFailedCreate(ctx, req, start, err)
	}
	remote, err := r.waitCreated(ctx, waiter)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

// waitCreated waits for the create operation of a database that a create request returned.
func (r *ResourcePostgresDatabase) waitCreated(ctx context.Context, waiter postgres.CreateDatabaseOperationInterface) (*PostgresDatabaseRemote, error) {
	result, err := waitCreateOperation(ctx, waiter.Wait)
	if err != nil {
		return nil, err
	}
	return makePostgresDatabaseRemote(result), nil
}

// recoverFailedCreate handles a create request that failed; see recoverCreate.
func (r *ResourcePostgresDatabase) recoverFailedCreate(ctx context.Context, req postgres.CreateDatabaseRequest, start time.Time, createErr error) (string, *PostgresDatabaseRemote, error) {
	if req.DatabaseId == "" {
		return "", nil, createErr
	}
	// A database found after a replace may predate it, and repeating a replace is safe.
	if req.ReplaceExisting {
		return "", nil, retrySafe(createErr)
	}

	name := req.Parent + "/databases/" + req.DatabaseId
	remote, err := recoverCreate(ctx, createRecovery[postgres.Database, *PostgresDatabaseRemote]{
		name:  "database " + name,
		start: start,
		get: func(ctx context.Context) (*postgres.Database, error) {
			return r.client.Postgres.GetDatabase(ctx, postgres.GetDatabaseRequest{Name: name})
		},
		createTime:      func(found *postgres.Database) (time.Time, bool) { return sdkTime(found.CreateTime) },
		alreadyExists:   isPostgresAlreadyExists,
		createdByCaller: nil,
		use: func(_ context.Context, found *postgres.Database) (*PostgresDatabaseRemote, error) {
			return makePostgresDatabaseRemote(found), nil
		},
		create: func(ctx context.Context) (func() (*PostgresDatabaseRemote, error), error) {
			waiter, err := r.client.Postgres.CreateDatabase(ctx, req)
			if err != nil {
				return nil, err
			}
			return func() (*PostgresDatabaseRemote, error) { return r.waitCreated(ctx, waiter) }, nil
		},
	}, createErr)
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
