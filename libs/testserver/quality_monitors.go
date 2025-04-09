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

	if checkExists {
		_, ok := s.monitors[tableName]
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

	s.monitors[tableName] = info
	return Response{
		Body: info,
	}
}

func (s *FakeWorkspace) QualityMonitorGet(req Request, tableName string) Response {
	info, ok := s.monitors[tableName]

	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	return Response{
		Body: info,
	}
}

func (s *FakeWorkspace) QualityMonitorDelete(req Request, tableName string) Response {
	_, ok := s.monitors[tableName]

	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	delete(s.monitors, tableName)
	return Response{}
}
