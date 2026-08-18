package direct

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// These calls bypass the SDK because it cannot read the response: it types sequence_id as
// an int64 while the service sends a JSON string, so a CreateOperation response fails to
// unmarshal. TODO(DMS): drop this file once the OpenAPI spec types the field as a string.

// operationResponse is the part of an operation response the CLI reads back.
type operationResponse struct {
	// SequenceId is the concurrency token for the next update, typed as the service sends it.
	SequenceId string `json:"sequence_id,omitempty"`
}

// updateOperationRequest carries the fields a later write for the same resource changes.
// action_type and resource_key are left out: the service fixes them at creation.
type updateOperationRequest struct {
	State        string                            `json:"state,omitempty"`
	ErrorMessage string                            `json:"error_message,omitempty"`
	ResourceId   string                            `json:"resource_id,omitempty"`
	Status       bundledeployments.OperationStatus `json:"status,omitempty"`
	SequenceId   string                            `json:"sequence_id,omitempty"`
}

// operationClient records operations under a deployment version. UpdateOperation
// takes the fields to update, because a failure updates fewer of them than a state
// write does.
type operationClient interface {
	CreateOperation(ctx context.Context, parent, resourceKey string, op bundledeployments.Operation) (operationResponse, error)
	UpdateOperation(ctx context.Context, parent, resourceKey string, fields []string, body updateOperationRequest) (operationResponse, error)
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

func (a *apiOperationClient) UpdateOperation(ctx context.Context, parent, resourceKey string, fields []string, body updateOperationRequest) (operationResponse, error) {
	var result operationResponse
	path := fmt.Sprintf("/api/2.0/bundle/%s/operations/%s", parent, resourceKey)
	err := a.client.Do(ctx, http.MethodPatch, path,
		auth.WorkspaceIDHeaders(a.client.Config),
		map[string]any{"update_mask": strings.Join(fields, ",")},
		body, &result)
	if err != nil {
		return operationResponse{}, err
	}
	return result, nil
}
