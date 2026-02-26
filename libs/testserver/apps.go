package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/apps"
)

func (s *FakeWorkspace) AppsCreateUpdate(req Request, name string) Response {
	var updateReq apps.AsyncUpdateAppRequest
	if err := json.Unmarshal(req.Body, &updateReq); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	existing, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	if updateReq.App != nil {
		// Convert both to maps and apply only the fields listed in update_mask.
		existingJSON, err := json.Marshal(existing)
		if err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		var existingMap map[string]any
		if err := json.Unmarshal(existingJSON, &existingMap); err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}

		updateJSON, err := json.Marshal(updateReq.App)
		if err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		var updateMap map[string]any
		if err := json.Unmarshal(updateJSON, &updateMap); err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}

		for _, field := range strings.Split(updateReq.UpdateMask, ",") {
			if v, ok := updateMap[strings.TrimSpace(field)]; ok {
				existingMap[strings.TrimSpace(field)] = v
			}
		}

		merged, err := json.Marshal(existingMap)
		if err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
		if err := json.Unmarshal(merged, &existing); err != nil {
			return Response{Body: fmt.Sprintf("internal error: %s", err), StatusCode: http.StatusInternalServerError}
		}
	}
	s.Apps[name] = existing

	return Response{
		Body: apps.AppUpdate{
			Status: &apps.AppUpdateUpdateStatus{
				State: apps.AppUpdateUpdateStatusUpdateStateSucceeded,
			},
		},
	}
}

func (s *FakeWorkspace) AppsGetUpdate(_ Request, name string) Response {
	defer s.LockUnlock()()

	_, ok := s.Apps[name]
	if !ok {
		return Response{StatusCode: 404}
	}

	return Response{
		Body: apps.AppUpdate{
			Status: &apps.AppUpdateUpdateStatus{
				State: apps.AppUpdateUpdateStatusUpdateStateSucceeded,
			},
		},
	}
}

func (s *FakeWorkspace) AppsUpsert(req Request, name string) Response {
	var app apps.App

	if err := json.Unmarshal(req.Body, &app); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	defer s.LockUnlock()()

	if name != "" {
		_, ok := s.Apps[name]
		if !ok {
			return Response{
				StatusCode: 404,
			}
		}
	} else {
		name = app.Name
		if name == "" {
			return Response{
				StatusCode: 400,
				Body:       "name is required",
			}
		}
		// Check if app already exists on create
		if _, exists := s.Apps[name]; exists {
			return Response{
				StatusCode: 409,
				Body: map[string]string{
					"error_code": "RESOURCE_ALREADY_EXISTS",
					"message":    "An app with the same name already exists: " + name,
				},
			}
		}
	}

	app.AppStatus = &apps.ApplicationStatus{
		State:   "RUNNING",
		Message: "Application is running.",
	}

	app.ComputeStatus = &apps.ComputeStatus{
		State:   "ACTIVE",
		Message: "App compute is active.",
	}

	app.Url = name + "-123.cloud.databricksapps.com"
	app.Id = strconv.Itoa(len(s.Apps) + 1000)

	s.Apps[name] = app
	return Response{
		Body: app,
	}
}
