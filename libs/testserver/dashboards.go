package testserver

import (
	"encoding/json"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/google/uuid"
)

func (s *FakeWorkspace) DashboardGet(dashboardId string) Response {
	defer s.LockUnlock()()

	value, ok := s.Dashboards[dashboardId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}
	return Response{
		Body: value,
	}
}

func (s *FakeWorkspace) DashboardCreate(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	dashboard.DashboardId = uuid.New().String()
	s.Dashboards[dashboard.DashboardId] = dashboard

	return Response{
		Body: dashboards.Dashboard{
			DashboardId: dashboard.DashboardId,
			Etag:        uuid.New().String(),
		},
	}
}

func (s *FakeWorkspace) DashboardUpdate(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	// Update the etag for the dashboard.
	dashboard.Etag = uuid.New().String()

	dashboardId := req.Vars["dashboard_id"]
	s.Dashboards[dashboardId] = dashboard

	return Response{
		Body: dashboard,
	}
}

func (s *FakeWorkspace) DashboardPublish(req Request) Response {
	defer s.LockUnlock()()

	var dashboard dashboards.Dashboard
	if err := json.Unmarshal(req.Body, &dashboard); err != nil {
		return Response{
			StatusCode: 400,
		}
	}

	return Response{
		Body: dashboards.PublishedDashboard{
			WarehouseId: dashboard.WarehouseId,
		},
	}
}
