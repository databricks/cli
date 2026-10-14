package testserver

import (
	qualitymonitors "github.com/databricks/cli/libs/qualitymonitors"
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *FakeWorkspace) QualityMonitorUpsert(req Request, tableName string, checkExists bool) Response {
	var request qualitymonitors.CreateMonitor
	var info qualitymonitors.MonitorInfo

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
