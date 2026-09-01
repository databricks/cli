package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/fuse"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
)

// registerFuseCredentials registers this server process with the node's FUSE
// daemons, so that /Workspace, /Volumes and /dbfs keep working for the server and
// for every session under it once the bootstrap notebook has exited.
//
// The registration is renewed in the background until ctx is cancelled.
func registerFuseCredentials(ctx context.Context, client *databricks.WorkspaceClient) error {
	self, err := fuse.Self()
	if err != nil {
		return fmt.Errorf("failed to describe this process: %w", err)
	}

	fuseClient, err := fuse.NewClient()
	if err != nil {
		return err
	}

	// The user id is only an audit-logging tag, so a workspace that will not answer
	// must not stop the registration itself.
	userID := ""
	if me, err := client.CurrentUser.Me(ctx, iam.MeRequest{}); err != nil {
		log.Warnf(ctx, "Failed to look up the current user, registering with the FUSE daemons without a user id: %v", err)
	} else {
		userID = me.Id
	}

	return fuse.KeepRegistered(ctx, fuseClient, self, workspaceToken(client), userID)
}

// workspaceToken returns the bearer token the resolved credentials currently
// authenticate with. It goes through Authenticate rather than reading Config.Token
// so that every credential type the CLI supports works, and so that a refreshed
// OAuth token is picked up on each call.
func workspaceToken(client *databricks.WorkspaceClient) fuse.TokenFunc {
	return func(ctx context.Context) (string, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, client.Config.Host, nil)
		if err != nil {
			return "", fmt.Errorf("failed to build a request to authenticate: %w", err)
		}
		if err := client.Config.Authenticate(req); err != nil {
			return "", fmt.Errorf("failed to authenticate: %w", err)
		}

		token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
		if !ok {
			return "", fmt.Errorf("resolved credentials are not a bearer token: %q", client.Config.AuthType)
		}
		return token, nil
	}
}
