package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/fuse"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

func registerFuseCredentials(ctx context.Context, client *databricks.WorkspaceClient) error {
	self, err := fuse.Self()
	if err != nil {
		return err
	}
	fuseClient, err := fuse.NewClient(self)
	if err != nil {
		return err
	}
	return fuse.KeepRegistered(ctx, fuseClient, workspaceToken(client), fuseUserID(ctx, client))
}

func fuseUserID(ctx context.Context, client *databricks.WorkspaceClient) string {
	// The user ID is an optional audit tag. Limit its lookup before publishing the server port.
	userCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	me, err := client.CurrentUser.Me(userCtx, iam.MeRequest{})
	if err != nil {
		log.Debugf(ctx, "Registering filesystem credentials without a user ID: %v", err)
		return ""
	}
	return me.Id
}

func workspaceToken(client *databricks.WorkspaceClient) fuse.TokenFunc {
	return func(ctx context.Context) (string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.Config.Host, nil)
		if err != nil {
			return "", fmt.Errorf("failed to build authentication request: %w", err)
		}
		// Authenticate picks up refreshed credentials, unlike reading Config.Token directly.
		if err := client.Config.Authenticate(req); err != nil {
			return "", fmt.Errorf("failed to authenticate: %w", err)
		}
		token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			return "", errors.New("resolved credentials do not provide a bearer token")
		}
		return token, nil
	}
}
