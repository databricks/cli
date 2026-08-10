package direct

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// The CLI calls the operations API directly rather than through the generated
// client because the SDK cannot read the response: it types sequence_id as an
// int64, while the service sends it as a JSON string (proto3 encodes 64-bit ints
// that way), so unmarshalling a CreateOperation response fails with
// "invalid character '1' after top-level value". The write itself succeeds - the
// status is 200 - so only the response parse is affected.
//
// TODO(DMS): this whole file goes away once the SDK types sequence_id as a string.
// The fix belongs in the OpenAPI spec the SDK is generated from, not here; until then
// every other DMS call still goes through the SDK, so keep the bypass to operations.

// operationResponse is the part of an operation response the CLI reads back.
type operationResponse struct {
	// SequenceId is the concurrency token for the next update of this operation.
	// Typed as a string because that is what the service sends; see above.
	SequenceId string `json:"sequence_id,omitempty"`
}

// updateOperationRequest carries the fields a later write for the same resource
// changes. action_type and resource_key are omitted: the service fixes them when
// the operation is created and ignores them here.
type updateOperationRequest struct {
	State        *json.RawMessage                  `json:"state,omitempty"`
	ErrorMessage string                            `json:"error_message,omitempty"`
	ResourceId   string                            `json:"resource_id,omitempty"`
	Status       bundledeployments.OperationStatus `json:"status,omitempty"`
	SequenceId   string                            `json:"sequence_id,omitempty"`
}

// operationClient records operations under a deployment version.
type operationClient interface {
	CreateOperation(ctx context.Context, parent, resourceKey string, op bundledeployments.Operation) (operationResponse, error)
	UpdateOperation(ctx context.Context, parent, resourceKey string, body updateOperationRequest) (operationResponse, error)
}

// apiOperationClient talks to the operations API through the workspace client.
type apiOperationClient struct {
	client *client.DatabricksClient
}

// newAPIOperationClient returns an operationClient that posts to the DMS API.
func newAPIOperationClient(c *client.DatabricksClient) operationClient {
	return &apiOperationClient{client: c}
}

func (a *apiOperationClient) CreateOperation(ctx context.Context, parent, resourceKey string, op bundledeployments.Operation) (operationResponse, error) {
	var result operationResponse
	path := fmt.Sprintf("/api/2.0/bundle/%s/operations", parent)
	err := a.client.Do(ctx, http.MethodPost, path,
		auth.WorkspaceIDHeaders(a.client.Config),
		map[string]any{"resource_key": resourceKey},
		op, &result)
	if err != nil {
		return operationResponse{}, err
	}
	return result, nil
}

func (a *apiOperationClient) UpdateOperation(ctx context.Context, parent, resourceKey string, body updateOperationRequest) (operationResponse, error) {
	var result operationResponse
	path := fmt.Sprintf("/api/2.0/bundle/%s/operations/%s", parent, resourceKey)
	err := a.client.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(a.client.Config),
		map[string]any{"update_mask": strings.Join(updatableFields, ",")},
		body, &result)
	if err != nil {
		return operationResponse{}, err
	}
	return result, nil
}
