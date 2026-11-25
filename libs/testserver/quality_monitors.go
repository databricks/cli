package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerRoutes("quality_monitors", addQualityMonitorsRoutes)
}

func addQualityMonitorsRoutes(server *Server) {
	// Quality monitors:

	// Simple handler for table existence check - returns a basic table info
	server.Handle("GET", "/api/2.1/unity-catalog/tables/{full_name}", func(req Request) any {
		parts := strings.Split(req.Vars["full_name"], ".")
		if len(parts) != 3 {
			return Response{StatusCode: 400, Body: "Invalid table name"}
		}
		return Response{
			Body: catalog.TableInfo{
				CatalogName: parts[0],
				SchemaName:  parts[1],
				Name:        parts[2],
				FullName:    req.Vars["full_name"],
			},
		}
	})

	server.Handle("GET", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return MapGet(req.Workspace, req.Workspace.Monitors, req.Vars["table_name"])
	})

	server.Handle("POST", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return req.Workspace.QualityMonitorUpsert(req, req.Vars["table_name"], false)
	})

	server.Handle("PUT", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return req.Workspace.QualityMonitorUpsert(req, req.Vars["table_name"], true)
	})

	server.Handle("DELETE", "/api/2.1/unity-catalog/tables/{table_name}/monitor", func(req Request) any {
		return MapDelete(req.Workspace, req.Workspace.Monitors, req.Vars["table_name"])
	})
}

func (s *FakeWorkspace) QualityMonitorUpsert(req Request, tableName string, checkExists bool) Response {
	var request catalog.CreateMonitor
	var info catalog.MonitorInfo

	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	err := jsonConvert(request, &info)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Cannot convert request to MonitorInfo: %s", err),
		}
	}

	defer s.LockUnlock()()

	if checkExists {
		existing, ok := s.Monitors[tableName]
		if !ok {
			return Response{
				StatusCode: 404,
			}
		}
		// For updates, preserve fields that cannot be changed after creation
		info.AssetsDir = existing.AssetsDir
	}

	if info.Status == "" {
		info.Status = "MONITOR_STATUS_ACTIVE"
	}

	if info.TableName == "" {
		info.TableName = tableName
	}

	if info.DriftMetricsTableName == "" {
		info.DriftMetricsTableName = tableName + "_drift_metrics"
	}

	if info.ProfileMetricsTableName == "" {
		info.ProfileMetricsTableName = tableName + "_profile_metrics"
	}

	if info.DashboardId == "" {
		info.DashboardId = nextDashboardID()
	}

	s.Monitors[tableName] = info
	return Response{
		Body: info,
	}
}
