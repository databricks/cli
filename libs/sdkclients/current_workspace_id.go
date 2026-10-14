package sdkclients

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/sdk-go/core/apierr"
)

// CurrentWorkspaceID returns the ID of the workspace the clients talk to. The
// SCIM Me endpoint reports it in the X-Databricks-Org-Id response header.
func (c *Clients) CurrentWorkspaceID(ctx context.Context) (int64, error) {
	url := strings.TrimSuffix(c.Config.CanonicalHostName(), "/") + "/api/2.0/preview/scim/v2/Me?excludedAttributes=entitlements"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if id := c.Config.WorkspaceID; id != "" {
		req.Header.Set("X-Databricks-Workspace-Id", id)
	}
	if err := c.Config.Authenticate(req); err != nil {
		return 0, err
	}
	resp, err := apiclient.HTTPClient(c.Config).Do(req)
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
