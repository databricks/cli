package iamutil

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// meRetryTimeout bounds how long GetCurrentUser retries transient 500s.
const meRetryTimeout = 30 * time.Second

// GetCurrentUser returns the current user, retrying on HTTP 500 responses.
//
// CurrentUser.Me is a read-only endpoint, so retrying any 500 is safe. The
// Databricks SCIM API intermittently returns "500 TEMPORARILY_UNAVAILABLE" (a
// condition it otherwise serves as 503), and the SDK retries 503 but not 500 —
// see the "some API's recommend retries on HTTP 500, but we'll add that later"
// comment on (*apierr.APIError).IsRetriable in databricks-sdk-go/apierr. This
// call runs at the start of every bundle command, so an unretried 500 fails the
// whole command; we close that gap here.
func GetCurrentUser(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return retries.Poll(ctx, meRetryTimeout, func() (*iam.User, *retries.Err) {
		user, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
		if err == nil {
			return user, nil
		}
		var apiErr *apierr.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusInternalServerError {
			return nil, retries.Continue(err)
		}
		return nil, retries.Halt(err)
	})
}
