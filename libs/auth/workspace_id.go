package auth

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/sdk-go/core/apierr"
)

// ResolveWorkspaceID returns the workspace ID as a string, preferring the
// value already configured on the client and falling back to a /Me probe.
//
// The fast path short-circuits the API call when w.Config.WorkspaceID is
// set (via --workspace-id, DATABRICKS_WORKSPACE_ID, workspace_id in
// .databrickscfg, ?o=/?w= on a host URL, or any other config source). The
// CLI-only "none" sentinel is treated as unset so it never leaks as a
// routing identifier.
//
// The fallback path reads the X-Databricks-Org-Id response header on
// /api/2.0/preview/scim/v2/Me and requires it to be an int64; the helper
// returns it as a string so callers can pass it to URL builders, env vars,
// and other string-typed sinks.
func ResolveWorkspaceID(ctx context.Context, w *sdkclients.Clients) (string, error) {
	if id := w.Config.WorkspaceID; id != "" && id != profile.WorkspaceIDNone {
		return id, nil
	}
	id, err := currentWorkspaceID(ctx, w)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

// currentWorkspaceID asks /Me for the workspace ID. It is a raw call because
// the ID is only in a response header, which sdk-go does not expose.
func currentWorkspaceID(ctx context.Context, w *sdkclients.Clients) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.Config.CanonicalHostName()+"/api/2.0/preview/scim/v2/Me?excludedAttributes=entitlements", nil)
	if err != nil {
		return 0, err
	}
	for k, v := range WorkspaceIDHeaders(w.Config) {
		req.Header.Set(k, v)
	}
	if err := w.Config.Authenticate(req); err != nil {
		return 0, err
	}
	resp, err := apiclient.HTTPClient(w.Config).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if apiErr := apierr.FromHTTPError(resp.StatusCode, resp.Header, body); apiErr != nil {
		return 0, apiErr
	}
	return strconv.ParseInt(resp.Header.Get("X-Databricks-Org-Id"), 10, 64)
}
