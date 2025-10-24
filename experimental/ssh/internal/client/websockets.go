package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/databricks/databricks-sdk-go"
	"github.com/gorilla/websocket"
)

func createWebsocketConnection(ctx context.Context, client *databricks.WorkspaceClient, connID, clusterID, publicKeyName string, serverPort int) (*websocket.Conn, error) {
	url, err := getProxyURL(ctx, client, connID, clusterID, publicKeyName, serverPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
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

func getProxyURL(ctx context.Context, client *databricks.WorkspaceClient, connID, clusterID, publicKeyName string, serverPort int) (string, error) {
	workspaceID, err := client.CurrentWorkspaceID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current workspace ID: %w", err)
	}
	u, err := url.Parse(client.Config.Host)
	if err != nil {
		return "", fmt.Errorf("failed to parse host URL: %w", err)
	}
	u.Path = fmt.Sprintf("/driver-proxy-api/o/%d/%s/%d/ssh", workspaceID, clusterID, serverPort)
	query := u.Query()
	query.Set("id", connID)
	query.Set("keyName", publicKeyName)
	u.RawQuery = query.Encode()
	return u.String(), nil
}
