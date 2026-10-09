package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// ProvisionedURLResponse represents the response from the account "primary
// provisioned URL" endpoint at
// /api/2.0/accounts/{account_id}/provisioned-urls/primary. The primary
// provisioned URL is the account's SPOG (Single Pane of Glass) host.
type ProvisionedURLResponse struct {
	URL string `json:"url"`
}

// WorkspacePrimaryURLResponse is the subset of a workspace host's
// /.well-known/databricks-config response that carries the owning account's
// primary (SPOG) URL. PrimaryURL is only returned when the request sets
// include_primary_url=true.
type WorkspacePrimaryURLResponse struct {
	PrimaryURL  string `json:"primary_url"`
	WorkspaceID string `json:"workspace_id"`
}

// LookupPrimaryProvisionedURL looks up the account's primary provisioned URL
// (its SPOG host) by account ID. It calls
// /api/2.0/accounts/{account_id}/provisioned-urls/primary on the given host
// using the supplied access token. Returns an error if the request fails or
// the response cannot be parsed. Callers should treat errors as non-fatal
// (best-effort profile enrichment).
func LookupPrimaryProvisionedURL(ctx context.Context, host, accountID, accessToken string, httpClient *http.Client) (string, error) {
	endpoint := strings.TrimSuffix(host, "/") + "/api/2.0/accounts/" + url.PathEscape(accountID) + "/provisioned-urls/primary"
	var provisioned ProvisionedURLResponse
	err := getJSON(ctx, httpClient, endpoint, accessToken, "provisioned-urls", &provisioned)
	// Accounts without a primary URL return 404.
	if errors.Is(err, errNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return provisioned.URL, nil
}

// LookupWorkspacePrimaryURL looks up the primary (SPOG) URL of the account
// that owns the workspace at host, along with that workspace's ID. It calls the
// unauthenticated /.well-known/databricks-config?include_primary_url=true
// endpoint. PrimaryURL is empty when the account has no primary URL. Callers
// should treat errors as non-fatal (best-effort profile enrichment).
func LookupWorkspacePrimaryURL(ctx context.Context, host string, httpClient *http.Client) (*WorkspacePrimaryURLResponse, error) {
	endpoint := strings.TrimSuffix(host, "/") + "/.well-known/databricks-config?include_primary_url=true"
	var discovery WorkspacePrimaryURLResponse
	if err := getJSON(ctx, httpClient, endpoint, "", "databricks-config", &discovery); err != nil {
		return nil, err
	}
	return &discovery, nil
}

// getJSON issues a GET to endpoint and decodes the JSON response into out. An
// empty accessToken sends the request unauthenticated. name identifies the
// endpoint in error messages.
// errNotFound is returned by getJSON for a 404 response.
var errNotFound = errors.New("not found")

func getJSON(ctx context.Context, httpClient *http.Client, endpoint, accessToken, name string, out any) error {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("creating %s request: %w", name, err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling %s endpoint: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain the body so the underlying TCP connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("%s endpoint returned status %d: %w", name, resp.StatusCode, errNotFound)
		}
		return fmt.Errorf("%s endpoint returned status %d", name, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s response: %w", name, err)
	}
	return nil
}

// oauthServerMetadata is the subset of an OAuth authorization server metadata
// document needed to check which host serves a workspace's tokens.
type oauthServerMetadata struct {
	TokenEndpoint string `json:"token_endpoint"`
}

// LookupOAuthTokenEndpoint returns the token_endpoint served by the OAuth
// authorization server metadata document at discoveryURL.
func LookupOAuthTokenEndpoint(ctx context.Context, discoveryURL string, httpClient *http.Client) (string, error) {
	var metadata oauthServerMetadata
	if err := getJSON(ctx, httpClient, discoveryURL, "", "oauth-authorization-server", &metadata); err != nil {
		return "", err
	}
	return metadata.TokenEndpoint, nil
}
