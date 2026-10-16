package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/dataquality"
)

func dataQualityMonitorKey(objectType, objectID string) string {
	return objectType + "/" + objectID
}

// The SDK sends the Monitor body directly (not wrapped in "monitor").
func (s *FakeWorkspace) DataQualityMonitorCreate(req Request) Response {
	defer s.LockUnlock()()

	var monitor dataquality.Monitor
	if err := json.Unmarshal(req.Body, &monitor); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	key := dataQualityMonitorKey(monitor.ObjectType, monitor.ObjectId)
	if _, ok := s.DataQualityMonitors[key]; ok {
		return Response{
			StatusCode: http.StatusConflict,
			Body:       map[string]string{"error_code": "RESOURCE_ALREADY_EXISTS", "message": "Monitor already exists for " + key},
		}
	}

	setDataProfilingOutputs(monitor.DataProfilingConfig, 1)
	s.DataQualityMonitors[key] = monitor
	return Response{Body: monitor}
}

// update_mask is a query parameter; the fake applies both configs, which is
// what the mask the CLI sends covers.
func (s *FakeWorkspace) DataQualityMonitorUpdate(req Request, objectType, objectID string) Response {
	defer s.LockUnlock()()

	key := dataQualityMonitorKey(objectType, objectID)
	existing, ok := s.DataQualityMonitors[key]
	if !ok {
		return Response{
			StatusCode: http.StatusNotFound,
			Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": "Monitor not found for " + key},
		}
	}

	var incoming dataquality.Monitor
	if err := json.Unmarshal(req.Body, &incoming); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	version := int64(1)
	if existing.DataProfilingConfig != nil {
		version = existing.DataProfilingConfig.MonitorVersion + 1
	}
	existing.AnomalyDetectionConfig = incoming.AnomalyDetectionConfig
	existing.DataProfilingConfig = incoming.DataProfilingConfig
	setDataProfilingOutputs(existing.DataProfilingConfig, version)

	s.DataQualityMonitors[key] = existing
	return Response{Body: existing}
}

// setDataProfilingOutputs fills server-computed fields so tests exercise drift
// suppression of OUTPUT_ONLY fields.
func setDataProfilingOutputs(config *dataquality.DataProfilingConfig, version int64) {
	if config == nil {
		return
	}
	config.Status = dataquality.DataProfilingStatusDataProfilingStatusActive
	config.MonitorVersion = version
	if config.DashboardId == "" {
		config.DashboardId = nextUUID()
	}
}
