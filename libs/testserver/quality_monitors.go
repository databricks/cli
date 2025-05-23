package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/databricks/databricks-sdk-go/service/catalog"
)

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
		_, ok := s.Monitors[tableName]
		if !ok {
			return Response{
				StatusCode: 404,
			}
		}
	}

	if info.Status == "" {
		info.Status = "MONITOR_STATUS_ACTIVE"
	}

	if info.TableName == "" {
		info.TableName = tableName
	}

	s.Monitors[tableName] = info
	return Response{
		Body: info,
	}
}
