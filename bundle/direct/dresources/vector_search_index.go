package dresources

import (
	"context"
	"fmt"
	"github.com/databricks/cli/libs/apierrs"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/sdkclients"
	vectorsearch "github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkvectorsearch "github.com/databricks/sdk-go/vectorsearch/v1"
)

// deleteIndexTimeout caps the wait for an index deletion to complete.
// In practice deletion finishes in a minute or two, but worst case the
// embedding pipeline shutdown can stretch closer to ten minutes.
const deleteIndexTimeout = 15 * time.Minute

// pendingDeletionTimeout caps how long a create is retried while the backend
// still holds the index name from a preceding delete (see createIndex). Kept
// deliberately short so we fail fast rather than hang for the full
// deleteIndexTimeout: if nightlies still hit the race, raise it then.
const pendingDeletionTimeout = time.Minute

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

// VectorSearchIndexRemote is remote state. endpoint_uuid is looked up from the
// endpoint service since the index API itself doesn't return it.
type VectorSearchIndexRemote struct {
	vectorsearch.VectorIndex
	EndpointUuid string `json:"endpoint_uuid,omitempty"`
}

type ResourceVectorSearchIndex struct {
	client *sdkclients.Clients
}

func (*ResourceVectorSearchIndex) New(client *sdkclients.Clients) *ResourceVectorSearchIndex {
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
			// pipeline_id is output-only and not part of the config.
		}
	}
	return state
}

func (r *ResourceVectorSearchIndex) getIndex(ctx context.Context, name string) (*sdkvectorsearch.VectorIndex, error) {
	return r.client.VectorSearch.GetVectorIndex(ctx, sdkvectorsearch.GetVectorIndexRequest{Name: new(name)})
}

func (r *ResourceVectorSearchIndex) DoRead(ctx context.Context, id string) (*VectorSearchIndexRemote, error) {
	resp, err := r.getIndex(ctx, id)
	if err != nil {
		return nil, err
	}
	index, err := vectorsearch.VectorIndexFromSDK(resp)
	if err != nil {
		return nil, err
	}
	endpointUuid, err := r.lookupEndpointUuid(ctx, utils.Deref(index.EndpointName))
	if err != nil {
		return nil, err
	}
	return &VectorSearchIndexRemote{
		VectorIndex:  *index,
		EndpointUuid: endpointUuid,
	}, nil
}

func (r *ResourceVectorSearchIndex) DoCreate(ctx context.Context, config *VectorSearchIndexState) (string, *VectorSearchIndexRemote, error) {
	request, err := vectorsearch.CreateVectorIndexRequestToSDK(&config.CreateVectorIndexRequest)
	if err != nil {
		return "", nil, err
	}
	resp, err := r.createIndex(ctx, *request)
	if err != nil {
		return "", nil, err
	}
	index, err := vectorsearch.VectorIndexFromSDK(resp)
	if err != nil {
		return "", nil, err
	}
	// Second API call (also done in DoRead): the index API does not return the
	// endpoint UUID, but we need to persist it in state so a future plan can
	// detect that the endpoint was replaced out-of-band (same name, different
	// UUID -> orphan).
	endpointUuid, err := r.lookupEndpointUuid(ctx, utils.Deref(config.EndpointName))
	if err != nil {
		return "", nil, err
	}
	config.EndpointUuid = endpointUuid
	return utils.Deref(config.Name), &VectorSearchIndexRemote{VectorIndex: *index, EndpointUuid: endpointUuid}, nil
}

// createIndex calls CreateIndex, retrying while the backend still reports the
// name as pending deletion.
//
// Deleting an index completes in two backend phases: the index first disappears
// from GET, and only later is the name released for reuse. WaitAfterDelete polls
// GET, so it can return as soon as phase one is done and a follow-up CREATE for
// the same name is still rejected with 400 INVALID_PARAMETER_VALUE "currently
// pending deletion". Polling GET cannot close that race because the two phases
// are observed on different endpoints, so the CREATE itself has to be retried.
// The API exposes no DELETING state to poll for instead; remove this once it
// does, or once CREATE queues behind the pending delete rather than failing.
func (r *ResourceVectorSearchIndex) createIndex(ctx context.Context, req sdkvectorsearch.CreateVectorIndexRequest) (*sdkvectorsearch.VectorIndex, error) {
	return retries.Poll(ctx, pendingDeletionTimeout, func() (*sdkvectorsearch.VectorIndex, *retries.Err) {
		index, err := r.client.VectorSearch.CreateVectorIndex(ctx, req)
		if err == nil {
			return index, nil
		}
		if isIndexPendingDeletion(err) {
			return nil, retries.Continues("index name is still pending deletion, waiting to recreate it")
		}
		return nil, retries.Halt(err)
	})
}

// isIndexPendingDeletion reports whether err is the backend's rejection of an
// operation on an index whose deletion has not fully completed. There is no
// distinct error code for it — the backend returns a generic
// INVALID_PARAMETER_VALUE — so the message has to be matched.
func isIndexPendingDeletion(err error) bool {
	if !apierrs.Is(err, apierrs.ErrInvalidParameterValue) {
		return false
	}
	return strings.Contains(apierrs.Message(err), "pending deletion")
}

// No DoUpdate: vector search indexes have no update API. All SDK fields are
// declared in vector_search_indexes.yml under recreate_on_changes or ignore_remote_changes.
// If a future SDK bump adds a new field that isn't classified, the framework
// rejects the resulting Update plan at bundle_plan.go (see also the reflection
// test in vector_search_index_test.go which catches it earlier at unit-test time).

func (r *ResourceVectorSearchIndex) DoDelete(ctx context.Context, id string, _ *VectorSearchIndexState) error {
	_, err := r.client.VectorSearch.DeleteVectorIndex(ctx, sdkvectorsearch.DeleteVectorIndexRequest{Name: new(id)})
	return err
}

// WaitAfterCreate polls GetIndex until Status.Ready=true. CreateIndex returns
// immediately with metadata of an index whose embedding pipeline is still
// provisioning; queries against an index that isn't ready fail. Blocking here
// lets dependent resources (and the next plan) see a usable index.
func (r *ResourceVectorSearchIndex) WaitAfterCreate(ctx context.Context, id string, config *VectorSearchIndexState) (*VectorSearchIndexRemote, error) {
	resp, err := retries.Poll(ctx, createIndexTimeout, func() (*sdkvectorsearch.VectorIndex, *retries.Err) {
		idx, getErr := r.getIndex(ctx, id)
		if getErr != nil {
			return nil, retries.Halt(getErr)
		}
		if idx.Status == nil || !utils.Deref(idx.Status.Ready) {
			msg := "index is still provisioning"
			if idx.Status != nil && utils.Deref(idx.Status.Message) != "" {
				msg = *idx.Status.Message
			}
			return nil, retries.Continues(msg)
		}
		return idx, nil
	})
	if err != nil {
		return nil, err
	}
	index, err := vectorsearch.VectorIndexFromSDK(resp)
	if err != nil {
		return nil, err
	}
	return &VectorSearchIndexRemote{VectorIndex: *index, EndpointUuid: config.EndpointUuid}, nil
}

// WaitAfterDelete polls GetIndex until the index is gone. The DELETE call is
// asynchronous, so without this a `bundle destroy` would report success while
// the index is still being torn down. The framework calls this after dropping
// state (so a wait-time failure leaves the bundle consistent) and maps the
// terminal NotFound to success (see Adapter.WaitAfterDelete).
//
// This does NOT on its own make a recreate safe: the name is released after the
// index disappears from GET, so createIndex retries the CREATE for the rest.
func (r *ResourceVectorSearchIndex) WaitAfterDelete(ctx context.Context, id string) error {
	_, err := retries.Poll[struct{}](ctx, deleteIndexTimeout, func() (*struct{}, *retries.Err) {
		_, getErr := r.getIndex(ctx, id)
		if getErr != nil {
			return nil, retries.Halt(getErr)
		}
		return nil, retries.Continues("index still exists, waiting for deletion to complete")
	})
	return err
}

// OverrideChangeDesc classifies endpoint_uuid drift: Recreate when the saved
// UUID differs from what's currently attached to the endpoint name, Skip
// otherwise. endpoint_uuid is never present in config, so without Skip a
// synthetic diff between empty newState and populated saved state would
// otherwise leak into the plan.
//
// Unlike vector_search_endpoint, this intentionally does NOT require
// remoteUuid != "". An empty remoteUuid here is the orphan signal: the index
// still exists by name but its backing endpoint has been deleted out-of-band.
// lookupEndpointUuid distinguishes this (404 -> "") from transient errors
// (propagated through DoRead/DoCreate), so reaching this branch with empty
// remoteUuid unambiguously means the endpoint is gone.
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
// name. A 404 is converted to ("", nil) so the caller can distinguish a
// genuinely missing endpoint (the orphan signal) from a transient or
// permission error, which is propagated.
func (r *ResourceVectorSearchIndex) lookupEndpointUuid(ctx context.Context, endpointName string) (string, error) {
	if endpointName == "" {
		return "", nil
	}
	info, err := r.client.VectorSearch.GetEndpoint(ctx, sdkvectorsearch.GetEndpointRequest{Name: new(endpointName)})
	if err != nil {
		if apierrs.IsMissing(err) {
			return "", nil
		}
		return "", fmt.Errorf("looking up vector search endpoint %q: %w", endpointName, err)
	}
	return utils.Deref(info.Id), nil
}
