package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ProvisionedURLResponse represents the response from the account "primary
// provisioned URL" endpoint at
// /api/2.0/accounts/{account_id}/provisioned-urls/primary. The primary
// provisioned URL is the account's SPOG (Single Pane of Glass) host.
type ProvisionedURLResponse struct {
	URL string `json:"url"`
}

// LookupPrimaryProvisionedURL looks up the account's primary provisioned URL
// (its SPOG host) by account ID. It calls
// /api/2.0/accounts/{account_id}/provisioned-urls/primary on the given host
// using the supplied access token. Returns an error if the request fails or
// the response cannot be parsed. Callers should treat errors as non-fatal
// (best-effort profile enrichment).
func LookupPrimaryProvisionedURL(ctx context.Context, host, accountID, accessToken string, httpClient *http.Client) (string, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	endpoint := strings.TrimSuffix(host, "/") + "/api/2.0/accounts/" + accountID + "/provisioned-urls/primary"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("creating provisioned-urls request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling provisioned-urls endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain the body so the underlying TCP connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return "", fmt.Errorf("provisioned-urls endpoint returned status %d", resp.StatusCode)
	}

	var provisioned ProvisionedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&provisioned); err != nil {
		return "", fmt.Errorf("decoding provisioned-urls response: %w", err)
	}
	return provisioned.URL, nil
}
