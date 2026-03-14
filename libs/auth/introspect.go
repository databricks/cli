package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// IntrospectionResponse represents the response from the Databricks token
// introspection endpoint at /api/2.0/tokens/introspect.
type IntrospectionResponse struct {
	PrincipalContext struct {
		AuthenticationScope struct {
			AccountID   string `json:"account_id"`
			WorkspaceID int64  `json:"workspace_id"`
		} `json:"authentication_scope"`
	} `json:"principal_context"`
}

// IntrospectionResult contains the extracted metadata from token introspection.
type IntrospectionResult struct {
	AccountID   string
	WorkspaceID string
}

// IntrospectToken calls the workspace token introspection endpoint to extract
// account_id and workspace_id for the given access token. Returns an error
// if the request fails or the response cannot be parsed. Callers should treat
// errors as non-fatal (best-effort metadata enrichment).
func IntrospectToken(ctx context.Context, host, accessToken string) (*IntrospectionResult, error) {
	endpoint := strings.TrimSuffix(host, "/") + "/api/2.0/tokens/introspect"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating introspection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling introspection endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Drain the body so the underlying TCP connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("introspection endpoint returned status %d", resp.StatusCode)
	}

	var introspection IntrospectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&introspection); err != nil {
		return nil, fmt.Errorf("decoding introspection response: %w", err)
	}

	result := &IntrospectionResult{
		AccountID: introspection.PrincipalContext.AuthenticationScope.AccountID,
	}
	if introspection.PrincipalContext.AuthenticationScope.WorkspaceID != 0 {
		result.WorkspaceID = strconv.FormatInt(introspection.PrincipalContext.AuthenticationScope.WorkspaceID, 10)
	}
	return result, nil
}
