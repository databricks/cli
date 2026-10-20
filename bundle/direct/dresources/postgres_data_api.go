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

// dataApiUpdateMask masks the whole spec. The direct engine sends the full
// desired spec on every update, so a field removed from the config resets to
// its server default. Masking only the changed leaves would instead be rejected
// for a removed field ("is in update_mask but not provided in request").
var dataApiUpdateMask = []string{"spec"}

// PostgresDataApiRemote is the return type for DoRead. It embeds
// DataApiDataApiSpec so that all paths in StateType are valid paths in
// RemoteType. GET does not return spec (it is INPUT_ONLY), so the embedded spec
// stays at its zero value and postgres_data_apis.generated.yml suppresses the
// resulting phantom drift via ignore_remote_changes with reason spec:input_only.
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
	var spec postgres.DataApiDataApiSpec
	if dataApi.Spec != nil {
		spec = *dataApi.Spec
	}
	return &PostgresDataApiRemote{
		DataApiDataApiSpec: spec,
		Parent:             dataApi.Parent,
		Name:               dataApi.Name,
		Status:             dataApi.Status,
		CreateTime:         dataApi.CreateTime,
		UpdateTime:         dataApi.UpdateTime,
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
	waiter, err := r.client.Postgres.CreateDataApi(ctx, postgres.CreateDataApiRequest{
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
	})
	if err != nil {
		return "", nil, err
	}

	result, err := waiter.Wait(ctx)
	if err != nil {
		return "", nil, err
	}

	remote := makePostgresDataApiRemote(result)
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
