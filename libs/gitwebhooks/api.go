// Package gitwebhooks calls the git-webhooks API, which holds the triggers that auto-deploy
// a bundle target on a push.
//
// The API is not in the public API spec yet, so the SDK does not generate it. The types here
// mirror the service's JSON.
package gitwebhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/listing"
)

const basePath = "/api/2.0/git-webhooks"

type API struct {
	client *databricksclient.DatabricksClient
}

func New(w *databricks.WorkspaceClient) (*API, error) {
	client, err := databricksclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &API{client: client}, nil
}

// UpsertWebhookTrigger creates the trigger, or updates the one with the same repository, bundle path
// and target.
func (a *API) UpsertWebhookTrigger(ctx context.Context, req UpsertWebhookTrigger) (*UpsertWebhookTriggerResponse, error) {
	var resp UpsertWebhookTriggerResponse
	err := a.client.Do(ctx, http.MethodPut, basePath, auth.WorkspaceIDHeaders(a.client.Config), nil, req, &resp)
	return &resp, err
}

// ListWebhookTriggers lists the triggers that the caller can view and that match the filter.
func (a *API) ListWebhookTriggers(ctx context.Context, req ListWebhookTriggersRequest) listing.Iterator[WebhookTrigger] {
	return listing.NewIterator(
		&req,
		a.listWebhookTriggersPage,
		func(resp *ListWebhookTriggersResponse) []WebhookTrigger {
			return resp.WebhookTriggers
		},
		func(resp *ListWebhookTriggersResponse) *ListWebhookTriggersRequest {
			if resp.NextPageToken == "" {
				return nil
			}
			req.PageToken = resp.NextPageToken
			return &req
		},
	)
}

func (a *API) listWebhookTriggersPage(ctx context.Context, req ListWebhookTriggersRequest) (*ListWebhookTriggersResponse, error) {
	var resp ListWebhookTriggersResponse
	err := a.client.Do(ctx, http.MethodGet, basePath+"/webhook-triggers", auth.WorkspaceIDHeaders(a.client.Config), nil, req, &resp)
	return &resp, err
}

// DeleteWebhookTrigger deletes the trigger and its runner job. Its attempts stay, and only
// workspace admins can read them.
func (a *API) DeleteWebhookTrigger(ctx context.Context, req DeleteWebhookTrigger) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	// The SDK sends the fields of a DELETE in the query string, but this endpoint reads them
	// from the body. So the request carries no data, and the visitor sets the body.
	return a.client.Do(ctx, http.MethodDelete, basePath+"/trigger", auth.WorkspaceIDHeaders(a.client.Config), nil, nil, nil, withJSONBody(body))
}

// withJSONBody returns a request visitor that sets body as the request's JSON body. The SDK runs
// visitors on every attempt, so each retry gets a fresh reader. GetBody lets the transport
// replay the request on a stale pooled connection, as it does for a body the SDK sets.
func withJSONBody(body []byte) func(*http.Request) error {
	return func(r *http.Request) error {
		r.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
		r.Body, _ = r.GetBody()
		r.ContentLength = int64(len(body))
		r.Header.Set("Content-Type", "application/json")
		return nil
	}
}
