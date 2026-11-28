// Package gitwebhooks calls the git-webhooks API, which holds the triggers that auto-deploy
// a bundle target on a push.
//
// The API is not in the public API spec yet, so the SDK does not generate it. The types here
// mirror the service's JSON.
package gitwebhooks

import (
	"context"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
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
