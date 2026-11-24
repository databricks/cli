package dresources

import (
	"context"
	"slices"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// dataApiUpdateMask masks the whole spec. With this mask the API sets every
// field the request carries and keeps the current value of a field it omits,
// so a field removed from the config keeps its last value. Leaf paths would
// reset an omitted field instead, but resetting db_max_rows or
// jwt_role_claim_key that way leaves the Data API unable to serve requests.
var dataApiUpdateMask = []string{"spec"}

// dataApiBoolFields are the spec's bool fields. Status omits a bool that is
// false and was never set, so the remote spec forces them: a missing bool in
// status means false, not unknown.
var dataApiBoolFields = []string{"DbAggregatesEnabled", "ServerTimingEnabled"}

// PostgresDataApiRemote is the return type for DoRead. It embeds
// DataApiDataApiSpec so that all paths in StateType are valid paths in
// RemoteType. GET does not return spec (it is INPUT_ONLY), so the embedded spec
// is filled from status, which reports the configuration in effect.
type PostgresDataApiRemote struct {
	postgres.DataApiDataApiSpec

	Parent string `json:"parent,omitempty"`

	Name       string                         `json:"name,omitempty"`
	Status     *postgres.DataApiDataApiStatus `json:"status,omitempty"`
	CreateTime *sdktime.Time                  `json:"create_time,omitempty"`
	UpdateTime *sdktime.Time                  `json:"update_time,omitempty"`
}

// Custom marshaler needed because embedded DataApiDataApiSpec has its own
// MarshalJSON which would otherwise take over and ignore the additional fields.
func (s *PostgresDataApiRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresDataApiRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourcePostgresDataApi struct {
	client *databricks.WorkspaceClient
}

type PostgresDataApiState = resources.PostgresDataApiConfig

func (*ResourcePostgresDataApi) New(client *databricks.WorkspaceClient) *ResourcePostgresDataApi {
	return &ResourcePostgresDataApi{client: client}
}

func (*ResourcePostgresDataApi) PrepareState(input *resources.PostgresDataApi) *PostgresDataApiState {
	return &PostgresDataApiState{
		DataApiDataApiSpec: input.DataApiDataApiSpec,
		Parent:             input.Parent,
	}
}

func (*ResourcePostgresDataApi) RemapState(remote *PostgresDataApiRemote) *PostgresDataApiState {
	return &PostgresDataApiState{
		DataApiDataApiSpec: remote.DataApiDataApiSpec,
		Parent:             remote.Parent,
	}
}

func makePostgresDataApiRemote(dataApi *postgres.DataApi) *PostgresDataApiRemote {
	return &PostgresDataApiRemote{
		DataApiDataApiSpec: dataApiSpecFromStatus(dataApi.Status),
		Parent:             dataApi.Parent,
		Name:               dataApi.Name,
		Status:             dataApi.Status,
		CreateTime:         dataApi.CreateTime,
		UpdateTime:         dataApi.UpdateTime,
	}
}

// dataApiSpecFromStatus returns the spec in effect, as reported by status. The
// spec on the resource cannot be used: GET omits it, and the one in an operation
// response leaves out the server defaults.
func dataApiSpecFromStatus(status *postgres.DataApiDataApiStatus) postgres.DataApiDataApiSpec {
	if status == nil {
		var spec postgres.DataApiDataApiSpec
		return spec
	}
	return postgres.DataApiDataApiSpec{
		DbAggregatesEnabled:      status.DbAggregatesEnabled,
		DbExtraSearchPath:        status.DbExtraSearchPath,
		DbMaxRows:                status.DbMaxRows,
		DbSchemas:                status.DbSchemas,
		JwtCacheMaxLifetime:      status.JwtCacheMaxLifetime,
		JwtRoleClaimKey:          status.JwtRoleClaimKey,
		OpenapiMode:              status.OpenapiMode,
		ServerCorsAllowedOrigins: status.ServerCorsAllowedOrigins,
		ServerTimingEnabled:      status.ServerTimingEnabled,
		ForceSendFields:          slices.Clone(dataApiBoolFields),
	}
}

func (r *ResourcePostgresDataApi) DoRead(ctx context.Context, id string) (*PostgresDataApiRemote, error) {
	dataApi, err := r.client.Postgres.GetDataApi(ctx, postgres.GetDataApiRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresDataApiRemote(dataApi), nil
}

// DoCreate enables the Data API on the parent database. If it is already
// enabled the API returns ALREADY_EXISTS; adopt it with `bundle deployment bind`.
func (r *ResourcePostgresDataApi) DoCreate(ctx context.Context, config *PostgresDataApiState) (string, *PostgresDataApiRemote, error) {
	req := postgres.CreateDataApiRequest{
		Parent: config.Parent,
		DataApi: postgres.DataApi{
			Spec: &config.DataApiDataApiSpec,

			// Output-only fields; the parent is carried in the request path.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
	}
	start := time.Now()
	waiter, err := r.client.Postgres.CreateDataApi(ctx, req)
	if err != nil {
		return r.recoverFailedCreate(ctx, req, start, err)
	}
	remote, err := r.waitCreated(ctx, waiter)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

// waitCreated waits for the create operation of a Data API that a create request returned.
func (r *ResourcePostgresDataApi) waitCreated(ctx context.Context, waiter postgres.CreateDataApiOperationInterface) (*PostgresDataApiRemote, error) {
	result, err := waitCreateOperation(ctx, waiter.Wait)
	if err != nil {
		return nil, err
	}
	return makePostgresDataApiRemote(result), nil
}

// recoverFailedCreate handles a create request that failed; see recoverCreate. A database has at
// most one Data API, named after the database.
func (r *ResourcePostgresDataApi) recoverFailedCreate(ctx context.Context, req postgres.CreateDataApiRequest, start time.Time, createErr error) (string, *PostgresDataApiRemote, error) {
	name := req.Parent + "/data-api"
	remote, err := recoverCreate(ctx, createRecovery[postgres.DataApi, *PostgresDataApiRemote]{
		name:  "data API " + name,
		start: start,
		get: func(ctx context.Context) (*postgres.DataApi, error) {
			return r.client.Postgres.GetDataApi(ctx, postgres.GetDataApiRequest{Name: name})
		},
		createTime: func(found *postgres.DataApi) (time.Time, bool) { return sdkTime(found.CreateTime) },
		use: func(_ context.Context, found *postgres.DataApi) (*PostgresDataApiRemote, error) {
			return makePostgresDataApiRemote(found), nil
		},
		create: func(ctx context.Context) (func() (*PostgresDataApiRemote, error), error) {
			waiter, err := r.client.Postgres.CreateDataApi(ctx, req)
			if err != nil {
				return nil, err
			}
			return func() (*PostgresDataApiRemote, error) { return r.waitCreated(ctx, waiter) }, nil
		},
	}, createErr)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func (r *ResourcePostgresDataApi) DoUpdate(ctx context.Context, id string, config *PostgresDataApiState, _ *PlanEntry) (*PostgresDataApiRemote, error) {
	waiter, err := r.client.Postgres.UpdateDataApi(ctx, postgres.UpdateDataApiRequest{
		Name: id,
		DataApi: postgres.DataApi{
			Spec: &config.DataApiDataApiSpec,

			// Output-only fields; the name is carried in the request path.
			CreateTime:      nil,
			Name:            "",
			Parent:          "",
			Status:          nil,
			UpdateTime:      nil,
			ForceSendFields: nil,
		},
		UpdateMask: fieldmask.FieldMask{
			Paths: dataApiUpdateMask,
		},
	})
	if err != nil {
		return nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return nil, err
	}
	return makePostgresDataApiRemote(result), nil
}

// DoDelete disables the Data API. The parent database and its data are not
// affected.
func (r *ResourcePostgresDataApi) DoDelete(ctx context.Context, id string, _ *PostgresDataApiState) error {
	waiter, err := r.client.Postgres.DeleteDataApi(ctx, postgres.DeleteDataApiRequest{
		Name: id,
	})
	if err != nil {
		return err
	}
	return waiter.Wait(ctx)
}
