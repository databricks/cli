package dresources

import (
	"context"
	"errors"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

// deleteIndexTimeout caps the wait for an index deletion to complete.
// In practice deletion finishes in a minute or two, but worst case the
// embedding pipeline shutdown can stretch closer to ten minutes.
const deleteIndexTimeout = 15 * time.Minute

// VectorSearchIndexState tracks the UUID of the endpoint the index is attached
// to. Without it the planner cannot tell that an index pointing at a deleted
// and recreated endpoint (same name, different UUID) has been orphaned — the
// index still exists by name but its backing endpoint is gone.
type VectorSearchIndexState struct {
	vectorsearch.CreateVectorIndexRequest
	EndpointUuid string `json:"endpoint_uuid,omitempty"`
}

func (s *VectorSearchIndexState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s VectorSearchIndexState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// VectorSearchIndexRemote is remote state. endpoint_uuid is looked up from the
// endpoint service since the index API itself doesn't return it.
type VectorSearchIndexRemote struct {
	*vectorsearch.VectorIndex
	EndpointUuid string `json:"endpoint_uuid,omitempty"`
}

func (s *VectorSearchIndexRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s VectorSearchIndexRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceVectorSearchIndex struct {
	client *databricks.WorkspaceClient
}

func (*ResourceVectorSearchIndex) New(client *databricks.WorkspaceClient) *ResourceVectorSearchIndex {
	return &ResourceVectorSearchIndex{client: client}
}

func (*ResourceVectorSearchIndex) PrepareState(input *resources.VectorSearchIndex) *VectorSearchIndexState {
	return &VectorSearchIndexState{
		CreateVectorIndexRequest: input.CreateVectorIndexRequest,
		EndpointUuid:             "",
	}
}

func (*ResourceVectorSearchIndex) RemapState(remote *VectorSearchIndexRemote) *VectorSearchIndexState {
	state := &VectorSearchIndexState{
		CreateVectorIndexRequest: vectorsearch.CreateVectorIndexRequest{
			DeltaSyncIndexSpec:    nil,
			DirectAccessIndexSpec: nil,
			IndexSubtype:          "",
			Name:                  remote.Name,
			EndpointName:          remote.EndpointName,
			IndexType:             remote.IndexType,
			PrimaryKey:            remote.PrimaryKey,
		},
		EndpointUuid: remote.EndpointUuid,
	}
	if remote.DeltaSyncIndexSpec != nil {
		state.DeltaSyncIndexSpec = &vectorsearch.DeltaSyncVectorIndexSpecRequest{
			ColumnsToSync:           nil,
			EmbeddingSourceColumns:  remote.DeltaSyncIndexSpec.EmbeddingSourceColumns,
			EmbeddingVectorColumns:  remote.DeltaSyncIndexSpec.EmbeddingVectorColumns,
			EmbeddingWritebackTable: remote.DeltaSyncIndexSpec.EmbeddingWritebackTable,
			PipelineType:            remote.DeltaSyncIndexSpec.PipelineType,
			SourceTable:             remote.DeltaSyncIndexSpec.SourceTable,
			ForceSendFields:         nil,
		}
	}
	if remote.DirectAccessIndexSpec != nil {
		state.DirectAccessIndexSpec = remote.DirectAccessIndexSpec
	}
	return state
}

func (r *ResourceVectorSearchIndex) DoRead(ctx context.Context, id string) (*VectorSearchIndexRemote, error) {
	index, err := r.client.VectorSearchIndexes.GetIndexByIndexName(ctx, id)
	if err != nil {
		return nil, err
	}
	return &VectorSearchIndexRemote{
		VectorIndex:  index,
		EndpointUuid: r.lookupEndpointUuid(ctx, index.EndpointName),
	}, nil
}

func (r *ResourceVectorSearchIndex) DoCreate(ctx context.Context, config *VectorSearchIndexState) (string, *VectorSearchIndexRemote, error) {
	index, err := r.client.VectorSearchIndexes.CreateIndex(ctx, config.CreateVectorIndexRequest)
	if err != nil {
		return "", nil, err
	}
	// Exceptional: a second API call. The index API does not return the endpoint
	// UUID, but we need to persist it in state so a future plan can detect that
	// the endpoint was replaced out-of-band (same name, different UUID -> orphan).
	endpointUuid := r.lookupEndpointUuid(ctx, config.EndpointName)
	config.EndpointUuid = endpointUuid
	return config.Name, &VectorSearchIndexRemote{VectorIndex: index, EndpointUuid: endpointUuid}, nil
}

func (r *ResourceVectorSearchIndex) DoUpdate(ctx context.Context, id string, config *VectorSearchIndexState, entry *PlanEntry) (*VectorSearchIndexRemote, error) {
	// Vector search indexes have no update API; all field changes trigger recreation via resources.yml.
	return nil, nil
}

func (r *ResourceVectorSearchIndex) DoDelete(ctx context.Context, id string) error {
	return r.client.VectorSearchIndexes.DeleteIndexByIndexName(ctx, id)
}

// WaitAfterDelete polls GetIndex until it returns 404. The DELETE call is
// asynchronous: a follow-up CREATE for the same name (e.g. during recreate) is
// rejected with "index is currently pending deletion" until the backend finishes
// tearing down the embedding pipeline. The framework calls this after dropping
// state so a wait-time failure leaves the bundle consistent.
func (r *ResourceVectorSearchIndex) WaitAfterDelete(ctx context.Context, id string) error {
	_, err := retries.Poll[struct{}](ctx, deleteIndexTimeout, func() (*struct{}, *retries.Err) {
		_, getErr := r.client.VectorSearchIndexes.GetIndexByIndexName(ctx, id)
		if getErr == nil {
			return nil, retries.Continues("index still exists, waiting for deletion to complete")
		}
		if errors.Is(getErr, apierr.ErrResourceDoesNotExist) || errors.Is(getErr, apierr.ErrNotFound) {
			return &struct{}{}, nil
		}
		return nil, retries.Halt(getErr)
	})
	return err
}

// OverrideChangeDesc classifies endpoint_uuid drift: Recreate when the saved
// UUID differs from what's currently attached to the endpoint name, Skip
// otherwise. endpoint_uuid is never present in config, so without Skip a
// synthetic diff between empty newState and populated saved state would
// otherwise leak into the plan.
func (*ResourceVectorSearchIndex) OverrideChangeDesc(_ context.Context, path *structpath.PathNode, change *ChangeDesc, remote *VectorSearchIndexRemote) error {
	if path.String() != "endpoint_uuid" {
		return nil
	}
	savedUuid, _ := change.Old.(string)
	var remoteUuid string
	if remote != nil {
		remoteUuid = remote.EndpointUuid
	}
	if savedUuid != "" && savedUuid != remoteUuid {
		change.Action = deployplan.Recreate
		change.Reason = "endpoint replaced out-of-band"
	} else {
		change.Action = deployplan.Skip
		change.Reason = "state-only field"
	}
	return nil
}

// lookupEndpointUuid returns the current UUID of the endpoint with the given
// name, or "" if the endpoint doesn't exist. Errors are logged and swallowed
// since a missing endpoint is the signal we want to capture in state.
func (r *ResourceVectorSearchIndex) lookupEndpointUuid(ctx context.Context, endpointName string) string {
	if endpointName == "" {
		return ""
	}
	info, err := r.client.VectorSearchEndpoints.GetEndpointByEndpointName(ctx, endpointName)
	if err != nil {
		if !apierr.IsMissing(err) {
			log.Warnf(ctx, "failed to read vector search endpoint %q while resolving index endpoint UUID: %v", endpointName, err)
		}
		return ""
	}
	return info.Id
}
