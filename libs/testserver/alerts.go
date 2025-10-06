package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"

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
