package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/gorilla/websocket"
)

func createWebsocketConnection(ctx context.Context, client *databricks.WorkspaceClient, connID, clusterID string, serverPort int, liteswap string) (*websocket.Conn, error) {
	url, err := getProxyURL(ctx, client, connID, clusterID, serverPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if liteswap != "" {
		req.Header.Set("x-databricks-traffic-id", "testenv://liteswap/"+liteswap)
	}
	if err := client.Config.Authenticate(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	req.URL.Scheme = "wss"
	// websocket connection manages lifecycle of the response object, no need to close the body
	conn, _, err := websocket.DefaultDialer.Dial(req.URL.String(), req.Header) // nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("failed to establish websocket connection: %w", err)
	}

	return conn, nil
}

func getProxyURL(ctx context.Context, client *databricks.WorkspaceClient, connID, clusterID string, serverPort int) (string, error) {
	workspaceID, err := auth.ResolveWorkspaceID(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get current workspace ID: %w", err)
	}
	host := client.Config.Host
	// The /driver-proxy-api/o/<workspace-id>/... path is a legacy URL form on
	// the driver-proxy endpoint and uses an "o" path segment regardless of
	// whether the workspace ID itself is the legacy or new shape.
	url := fmt.Sprintf("%s/driver-proxy-api/o/%s/%s/%d/ssh?id=%s", host, workspaceID, clusterID, serverPort, connID)
	return url, nil
}
