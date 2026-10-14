// Package sqllegacy is a client for the subset of the legacy Databricks SQL
// API (/api/2.0/preview/sql/...) that the CLI uses outside generated
// commands. sdk-go does not provide these endpoints.
package sqllegacy

import (
	"context"
	"net/http"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/listing"
)

// Client calls the legacy SQL API.
type Client struct {
	client *apiclient.Client
}

func New(c *apiclient.Client) *Client {
	return &Client{client: c}
}

func (a *Client) headers() map[string]string {
	headers := map[string]string{"Accept": "application/json"}
	a.client.AddWorkspaceIDHeader(headers)
	return headers
}

// ListDashboards pages through legacy dashboards, starting from page 1.
func (a *Client) ListDashboards(ctx context.Context, request ListDashboardsRequest) listing.Iterator[Dashboard] {
	request.Page = 1
	getNextPage := func(ctx context.Context, req ListDashboardsRequest) (*ListResponse, error) {
		var resp ListResponse
		err := a.client.Do(ctx, http.MethodGet, "/api/2.0/preview/sql/dashboards", a.headers(), nil, apiclient.URLQuery(req), &resp)
		return &resp, err
	}
	getItems := func(resp *ListResponse) []Dashboard {
		return resp.Results
	}
	getNextReq := func(resp *ListResponse) *ListDashboardsRequest {
		if len(resp.Results) == 0 {
			return nil
		}
		request.Page = resp.Page + 1
		return &request
	}
	return listing.NewIterator(&request, getNextPage, getItems, getNextReq)
}

// ListDashboardsAll returns all legacy dashboards, at most PageSize of them
// when it is set.
func (a *Client) ListDashboardsAll(ctx context.Context, request ListDashboardsRequest) ([]Dashboard, error) {
	return listing.ToSliceN[Dashboard, int](ctx, a.ListDashboards(ctx, request), request.PageSize)
}

// ListDataSources returns the SQL warehouses usable as legacy query data sources.
func (a *Client) ListDataSources(ctx context.Context) ([]DataSource, error) {
	var dataSources []DataSource
	err := a.client.Do(ctx, http.MethodGet, "/api/2.0/preview/sql/data_sources", a.headers(), nil, nil, &dataSources)
	return dataSources, err
}
