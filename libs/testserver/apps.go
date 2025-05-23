package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/apps"
)

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
