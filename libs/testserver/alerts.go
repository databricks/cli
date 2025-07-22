package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/google/uuid"
)

func (s *FakeWorkspace) AlertsUpsert(req Request, alertId string) Response {
	var alert sql.AlertV2

	if err := json.Unmarshal(req.Body, &alert); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	if alertId != "" {
		_, ok := s.Alerts[alertId]
		if !ok {
			return Response{
				StatusCode: 404,
			}
		}
	} else {
		alertId = uuid.New().String()
	}

	alert.Id = alertId
	alert.LifecycleState = sql.LifecycleStateActive
	s.Alerts[alertId] = alert

	return Response{
		StatusCode: 200,
		Body:       alert,
	}
}

func (s *FakeWorkspace) AlertsList(req Request) Response {
	defer s.LockUnlock()()

	var alerts []sql.AlertV2
	for _, alert := range s.Alerts {
		alerts = append(alerts, alert)
	}

	// Parse query parameters for pagination
	queryParams, _ := url.ParseQuery(req.URL.RawQuery)
	pageToken := queryParams.Get("page_token")
	maxResults := 50 // Default page size

	if maxResultsStr := queryParams.Get("max_results"); maxResultsStr != "" {
		if parsed, err := strconv.Atoi(maxResultsStr); err == nil {
			maxResults = parsed
		}
	}

	// Simple pagination simulation
	startIndex := 0
	if pageToken != "" {
		if parsed, err := strconv.Atoi(pageToken); err == nil {
			startIndex = parsed
		}
	}

	endIndex := startIndex + maxResults
	if endIndex > len(alerts) {
		endIndex = len(alerts)
	}

	pageAlerts := alerts[startIndex:endIndex]

	response := sql.ListAlertsV2Response{
		Results: pageAlerts,
	}

	// Set next page token if there are more results
	if endIndex < len(alerts) {
		response.NextPageToken = strconv.Itoa(endIndex)
	}

	return Response{
		StatusCode: 200,
		Body:       response,
	}
}
