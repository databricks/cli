// Code ported from github.com/databricks/databricks-sdk-go.

package qualitymonitors

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/libs/apiclient"
)

type qualityMonitorsImpl struct {
	client *apiclient.Client
}

func (a *qualityMonitorsImpl) CancelRefresh(ctx context.Context, request CancelRefreshRequest) error {
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor/refreshes/%v/cancel", request.TableName, request.RefreshId)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, nil)
	return err
}

func (a *qualityMonitorsImpl) Create(ctx context.Context, request CreateMonitor) (*MonitorInfo, error) {
	var monitorInfo MonitorInfo
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, &monitorInfo)
	return &monitorInfo, err
}

func (a *qualityMonitorsImpl) Delete(ctx context.Context, request DeleteQualityMonitorRequest) (*DeleteMonitorResponse, error) {
	var deleteMonitorResponse DeleteMonitorResponse
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodDelete, path, headers, queryParams, apiclient.URLQuery(request), &deleteMonitorResponse)
	return &deleteMonitorResponse, err
}

func (a *qualityMonitorsImpl) Get(ctx context.Context, request GetQualityMonitorRequest) (*MonitorInfo, error) {
	var monitorInfo MonitorInfo
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &monitorInfo)
	return &monitorInfo, err
}

func (a *qualityMonitorsImpl) GetRefresh(ctx context.Context, request GetRefreshRequest) (*MonitorRefreshInfo, error) {
	var monitorRefreshInfo MonitorRefreshInfo
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor/refreshes/%v", request.TableName, request.RefreshId)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &monitorRefreshInfo)
	return &monitorRefreshInfo, err
}

func (a *qualityMonitorsImpl) ListRefreshes(ctx context.Context, request ListRefreshesRequest) (*MonitorRefreshListResponse, error) {
	var monitorRefreshListResponse MonitorRefreshListResponse
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor/refreshes", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodGet, path, headers, queryParams, apiclient.URLQuery(request), &monitorRefreshListResponse)
	return &monitorRefreshListResponse, err
}

func (a *qualityMonitorsImpl) RegenerateDashboard(ctx context.Context, request RegenerateDashboardRequest) (*RegenerateDashboardResponse, error) {
	var regenerateDashboardResponse RegenerateDashboardResponse
	path := fmt.Sprintf("/api/2.1/quality-monitoring/tables/%v/monitor/dashboard", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, &regenerateDashboardResponse)
	return &regenerateDashboardResponse, err
}

func (a *qualityMonitorsImpl) RunRefresh(ctx context.Context, request RunRefreshRequest) (*MonitorRefreshInfo, error) {
	var monitorRefreshInfo MonitorRefreshInfo
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor/refreshes", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPost, path, headers, queryParams, request, &monitorRefreshInfo)
	return &monitorRefreshInfo, err
}

func (a *qualityMonitorsImpl) Update(ctx context.Context, request UpdateMonitor) (*MonitorInfo, error) {
	var monitorInfo MonitorInfo
	path := fmt.Sprintf("/api/2.1/unity-catalog/tables/%v/monitor", request.TableName)
	queryParams := make(map[string]any)
	headers := make(map[string]string)
	headers["Accept"] = "application/json"
	headers["Content-Type"] = "application/json"
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, http.MethodPut, path, headers, queryParams, request, &monitorInfo)
	return &monitorInfo, err
}
