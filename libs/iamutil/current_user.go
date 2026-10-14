package iamutil

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"github.com/databricks/sdk-go/core/apierr"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
)

const meRetryTimeout = 30 * time.Second

// meRetryInterval is the pause between attempts while the SCIM Me endpoint
// returns 500, which it does transiently right after workspace creation.
const meRetryInterval = time.Second

// GetCurrentUser returns the authenticated user. It is returned as the
// JSON-capable mirror type because callers store it in bundle configuration.
func GetCurrentUser(ctx context.Context, w *sdkclients.Clients) (*scim.User, error) {
	deadline := time.Now().Add(meRetryTimeout)
	for {
		user, err := w.Scim.Me(ctx, sdkscim.MeRequest{})
		if err == nil {
			return scim.UserFromSDK(user)
		}
		apiErr, ok := errors.AsType[*apierr.APIError](err)
		if !ok || apiErr.HTTPStatusCode() != http.StatusInternalServerError || time.Now().Add(meRetryInterval).After(deadline) {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(meRetryInterval):
		}
	}
}
