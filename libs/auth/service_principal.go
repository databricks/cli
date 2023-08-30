package auth

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

// Determines whether a given user id is a service principal.
// This function uses a heuristic: if no user exists with this id, we assume
// it's a service principal. Unfortunately, the standard service principal API is too
// slow for our purposes.
func IsServicePrincipal(ctx context.Context, ws *databricks.WorkspaceClient, userId string) bool {
	_, err := ws.Users.GetById(ctx, userId)
	return err != nil
}
