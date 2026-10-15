// Package qualitymonitors is a client for the legacy Unity Catalog quality
// monitors API (/api/2.1/unity-catalog/tables/{table_name}/monitor), which
// sdk-go does not provide (it only has the data-quality v1 API).
package qualitymonitors

import "github.com/databricks/cli/libs/apiclient"

// Client calls the quality monitors API.
type Client struct {
	qualityMonitorsImpl
}

func New(c *apiclient.Client) *Client {
	return &Client{qualityMonitorsImpl{client: c}}
}

func addWorkspaceID(c *apiclient.Client, headers map[string]string) {
	c.AddWorkspaceIDHeader(headers)
}
