package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	"github.com/gorilla/websocket"
)

func createWebsocketConnection(ctx context.Context, client *databricks.WorkspaceClient, connID, clusterID string, serverPort int, liteswap string) (*websocket.Conn, error) {
	proxyURL, err := getProxyURL(ctx, client, connID, clusterID, serverPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, proxyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if liteswap != "" {
		req.Header.Set("x-databricks-traffic-id", "testenv://liteswap/"+liteswap)
	}
	if err := client.Config.Authenticate(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

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
	return buildProxyWebsocketURL(client.Config.Host, workspaceID, clusterID, serverPort, connID)
}

// buildProxyWebsocketURL builds the driver-proxy websocket URL for an SSH tunnel.
//
// The scheme follows the host (http -> ws, else wss) instead of being hardcoded to
// wss, so the tunnel is also diallable against the plaintext local test server.
func buildProxyWebsocketURL(host, workspaceID, clusterID string, serverPort int, connID string) (string, error) {
	u, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("failed to parse host %q: %w", host, err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	default:
		u.Scheme = "wss"
	}
	// The /driver-proxy-api/o/<workspace-id>/... path is a legacy URL form on
	// the driver-proxy endpoint and uses an "o" path segment regardless of
	// whether the workspace ID itself is the legacy or new shape.
	u.Path = fmt.Sprintf("/driver-proxy-api/o/%s/%s/%d/ssh", workspaceID, clusterID, serverPort)
	u.RawQuery = url.Values{"id": {connID}}.Encode()
	return u.String(), nil
}
