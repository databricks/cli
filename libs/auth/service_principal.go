package auth

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// Determines whether a given user id is a service principal.
// This function uses a heuristic: if no user exists with this id, we assume
// it's a service principal. Unfortunately, the standard service principal API is too
// slow for our purposes.
func IsServicePrincipal(ctx context.Context, ws *databricks.WorkspaceClient, userId string) (bool, error) {
	_, err := ws.Users.GetById(ctx, userId)
	if userId == "" {
		// If the user id is empty (for testing) or a non-email address then
		// we can assume that it's not a service principal. We don't yet
		// rely on the latter since that's not officially documented.
		return false, nil
	}
	if apierr.IsMissing(err) {
		return true, nil
	}
	return false, err
}
