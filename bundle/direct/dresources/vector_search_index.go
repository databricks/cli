package dresources

import (
	"context"
	"errors"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
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

// createIndexTimeout caps the wait for an index to become ready after creation.
// Delta sync indexes do an initial sync from the source table, which can stretch
// out for large tables. Matches the terraform provider's defaultIndexProvisionTimeout.
// https://github.com/databricks/terraform-provider-databricks/blob/c79d82d9582ab6670468bbff303199906d47905f/vectorsearch/resource_vector_search_index.go#L19
const createIndexTimeout = 75 * time.Minute

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

// VectorSearchIndexRemote is remote state. endpoint_uuid mirrors the index
// response's endpoint_id so OverrideChangeDesc can compare it against the
// saved state without a second API call.
type VectorSearchIndexRemote struct {
	vectorsearch.VectorIndex
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
			DeltaSyncIndexSpec:    nil, // need to remap below
			DirectAccessIndexSpec: remote.DirectAccessIndexSpec,
			IndexSubtype:          remote.IndexSubtype,
			Name:                  remote.Name,
			EndpointName:          remote.EndpointName,
			IndexType:             remote.IndexType,
			PrimaryKey:            remote.PrimaryKey,
		},
		EndpointUuid: remote.EndpointUuid,
	}
	if remote.DeltaSyncIndexSpec != nil {
		state.DeltaSyncIndexSpec = &vectorsearch.DeltaSyncVectorIndexSpecRequest{
			ColumnsToIndex:          remote.DeltaSyncIndexSpec.ColumnsToIndex,
			ColumnsToSync:           remote.DeltaSyncIndexSpec.ColumnsToSync,
			EmbeddingSourceColumns:  remote.DeltaSyncIndexSpec.EmbeddingSourceColumns,
			EmbeddingVectorColumns:  remote.DeltaSyncIndexSpec.EmbeddingVectorColumns,
			EmbeddingWritebackTable: remote.DeltaSyncIndexSpec.EmbeddingWritebackTable,
			PipelineType:            remote.DeltaSyncIndexSpec.PipelineType,
			SourceTable:             remote.DeltaSyncIndexSpec.SourceTable,
			// ForceSendFields is an SDK marshaling concern (which zero-valued
			// fields to wire-serialize) that has no meaning on the read path.
			// Local config doesn't carry one either, so leave it nil rather
			// than copy whatever the response struct happened to use.
			ForceSendFields: nil,
		}
	}
	return state
}

func (r *ResourceVectorSearchIndex) DoRead(ctx context.Context, id string) (*VectorSearchIndexRemote, error) {
	index, err := r.client.VectorSearchIndexes.GetIndexByIndexName(ctx, id)
	if err != nil {
		return nil, err
	}
	return &VectorSearchIndexRemote{
		VectorIndex:  *index,
		EndpointUuid: index.EndpointId,
	}, nil
}

func (r *ResourceVectorSearchIndex) DoCreate(ctx context.Context, config *VectorSearchIndexState) (string, *VectorSearchIndexRemote, error) {
	index, err := r.client.VectorSearchIndexes.CreateIndex(ctx, config.CreateVectorIndexRequest)
	if err != nil {
		return "", nil, err
	}
	config.EndpointUuid = index.EndpointId
	return config.Name, &VectorSearchIndexRemote{VectorIndex: *index, EndpointUuid: index.EndpointId}, nil
}

// No DoUpdate: vector search indexes have no update API. All SDK fields are
// declared in resources.yml under recreate_on_changes or ignore_remote_changes.
// If a future SDK bump adds a new field that isn't classified, the framework
// rejects the resulting Update plan at bundle_plan.go (see also the reflection
// test in vector_search_index_test.go which catches it earlier at unit-test time).

func (r *ResourceVectorSearchIndex) DoDelete(ctx context.Context, id string, _ *VectorSearchIndexState) error {
	return r.client.VectorSearchIndexes.DeleteIndexByIndexName(ctx, id)
}

// WaitAfterCreate polls GetIndex until Status.Ready=true. CreateIndex returns
// immediately with metadata of an index whose embedding pipeline is still
// provisioning; queries against an index that isn't ready fail. Blocking here
// lets dependent resources (and the next plan) see a usable index.
func (r *ResourceVectorSearchIndex) WaitAfterCreate(ctx context.Context, id string, config *VectorSearchIndexState) (*VectorSearchIndexRemote, error) {
	index, err := retries.Poll(ctx, createIndexTimeout, func() (*vectorsearch.VectorIndex, *retries.Err) {
		idx, getErr := r.client.VectorSearchIndexes.GetIndexByIndexName(ctx, id)
		if getErr != nil {
			return nil, retries.Halt(getErr)
		}
		if idx.Status == nil || !idx.Status.Ready {
			msg := "index is still provisioning"
			if idx.Status != nil && idx.Status.Message != "" {
				msg = idx.Status.Message
			}
			return nil, retries.Continues(msg)
		}
		return idx, nil
	})
	if err != nil {
		return nil, err
	}
	return &VectorSearchIndexRemote{VectorIndex: *index, EndpointUuid: config.EndpointUuid}, nil
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
// UUID differs from the endpoint_id the index API now reports, Skip otherwise.
// endpoint_uuid is never present in config, so without Skip a synthetic diff
// between empty newState and populated saved state would leak into the plan.
//
// An empty remoteUuid is treated as drift (rather than ignored, as the endpoint
// resource does for its own UUID). If the backend ever clears endpoint_id when
// the endpoint is deleted out-of-band, this surfaces the orphan as a recreate.
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
