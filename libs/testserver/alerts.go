package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// alertFile mirrors the schema of a .dbalert.json file. It is the inverse of
// bundle/config/mutator/load_dbalert_files.go: API-only fields (display_name,
// warehouse_id) are dropped and the joined query_text/custom_description are
// split back into lines. The field order matches what the backend materializes.
type alertFile struct {
	CustomSummary          string                 `json:"custom_summary,omitempty"`
	Evaluation             *sql.AlertV2Evaluation `json:"evaluation,omitempty"`
	Schedule               *sql.CronSchedule      `json:"schedule,omitempty"`
	QueryLines             []string               `json:"query_lines,omitempty"`
	CustomDescriptionLines []string               `json:"custom_description_lines,omitempty"`
}

// alertFilePath returns the workspace path of the .dbalert.json file the backend
// materializes for an alert: <parent_path>/<display_name>.dbalert.json.
func alertFilePath(alert sql.AlertV2) string {
	return path.Join(alert.ParentPath, alert.DisplayName+".dbalert.json")
}

// splitLines reverses the line-joining done in load_dbalert_files.go: each line
// is terminated by "\n", so splitting drops the trailing empty element.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// writeAlertFile materializes the .dbalert.json file for an alert. On real cloud
// the backend writes this file as a side effect of alert creation/update, which
// the `workspace export` round-trip and `bundle generate alert` rely on.
func (s *FakeWorkspace) writeAlertFile(alert sql.AlertV2) error {
	if alert.ParentPath == "" || alert.DisplayName == "" {
		return nil
	}

	evaluation := alert.Evaluation
	if evaluation.Notification != nil {
		// The backend always serializes notify_on_ok in the file, even when
		// false; the SDK marshaler would otherwise drop the zero value.
		notification := *evaluation.Notification
		notification.ForceSendFields = append(notification.ForceSendFields, "NotifyOnOk")
		evaluation.Notification = &notification
	}

	af := alertFile{
		CustomSummary:          alert.CustomSummary,
		Evaluation:             &evaluation,
		Schedule:               &alert.Schedule,
		QueryLines:             splitLines(alert.QueryText),
		CustomDescriptionLines: splitLines(alert.CustomDescription),
	}

	data, err := json.MarshalIndent(af, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	filePath := alertFilePath(alert)
	s.files[filePath] = FileEntry{
		Info: workspace.ObjectInfo{
			ObjectType: "FILE",
			Path:       filePath,
			ObjectId:   nextID(),
		},
		Data: data,
	}
	return nil
}

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
		alertId = nextUUID()
	}

	alert.Id = alertId
	alert.LifecycleState = sql.AlertLifecycleStateActive
	s.Alerts[alertId] = alert

	if err := s.writeAlertFile(alert); err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: http.StatusInternalServerError,
		}
	}

	return Response{
		StatusCode: 200,
		Body:       alert,
	}
}

func (s *FakeWorkspace) AlertsDelete(alertId string, purge bool) Response {
	defer s.LockUnlock()()

	alert, ok := s.Alerts[alertId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	delete(s.files, alertFilePath(alert))

	if purge {
		delete(s.Alerts, alertId)
	} else {
		alert.LifecycleState = sql.AlertLifecycleStateDeleted
		s.Alerts[alertId] = alert
	}

	return Response{
		StatusCode: 200,
	}
}
