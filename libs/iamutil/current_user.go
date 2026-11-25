package iamutil

import (
	"context"

	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// GetCurrentUser returns the current user, retrying on the errors [retry.IsTransientRead]
// covers: 5xx responses the SDK does not retry itself and transport errors such as the server
// closing the connection during the TLS handshake ("EOF").
//
// CurrentUser.Me is a read-only endpoint, so retrying any of them is safe. The
// Databricks SCIM API intermittently returns "500 TEMPORARILY_UNAVAILABLE" (a
// condition it otherwise serves as 503), and the SDK retries 503 but not 500 —
// see the "some API's recommend retries on HTTP 500, but we'll add that later"
// comment on (*apierr.APIError).IsRetriable in databricks-sdk-go/apierr. This
// call runs at the start of every bundle command, so an unretried error fails the
// whole command; we close that gap here.
func GetCurrentUser(ctx context.Context, w *databricks.WorkspaceClient) (*iam.User, error) {
	return retry.With(ctx, retry.IsTransientRead, func() (*iam.User, error) {
		return w.CurrentUser.Me(ctx, iam.MeRequest{})
	})
}
