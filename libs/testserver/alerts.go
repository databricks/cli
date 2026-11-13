package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// alertFile mirrors the schema of a .dbalert.json file. It is the inverse of
// bundle/config/mutator/load_dbalert_files.go: API-only fields (display_name,
// warehouse_id) are dropped and the joined query_text/custom_description are
// split back into lines. The field order matches what the backend materializes.
type alertFile struct {
	CustomSummary          string                  `json:"custom_summary,omitempty"`
	Evaluation             *materializedEvaluation `json:"evaluation,omitempty"`
	Schedule               *sql.CronSchedule       `json:"schedule,omitempty"`
	QueryLines             []string                `json:"query_lines,omitempty"`
	CustomDescriptionLines []string                `json:"custom_description_lines,omitempty"`
}

// materializedEvaluation mirrors the field order the alerts backend uses when it
// writes the .dbalert.json file, which differs from sql.AlertV2Evaluation: the
// backend emits source before comparison_operator and always writes a
// notification object (an empty {} when none is configured).
type materializedEvaluation struct {
	Source             sql.AlertV2OperandColumn `json:"source"`
	ComparisonOperator sql.ComparisonOperator   `json:"comparison_operator"`
	Threshold          *materializedOperand     `json:"threshold,omitempty"`
	Notification       *sql.AlertV2Notification `json:"notification"`
}

// materializedOperand and materializedOperandValue mirror sql.AlertV2Operand but
// format double_value via jsonFloat so it keeps its decimal point in the file.
type materializedOperand struct {
	Column *sql.AlertV2OperandColumn `json:"column,omitempty"`
	Value  *materializedOperandValue `json:"value,omitempty"`
}

type materializedOperandValue struct {
	BoolValue   bool       `json:"bool_value,omitempty"`
	DoubleValue *jsonFloat `json:"double_value,omitempty"`
	StringValue string     `json:"string_value,omitempty"`
}

// jsonFloat marshals a float64 the way the alerts backend serializes numeric
// thresholds in .dbalert.json: always with a decimal point (0 -> "0.0"). Go's
// encoding/json drops the fractional part for integer-valued floats.
type jsonFloat float64

func (f jsonFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}

// materializeThreshold converts the SDK threshold into the file representation.
// double_value is emitted whenever the source request set it explicitly (tracked
// via ForceSendFields), even when it is the zero value.
func materializeThreshold(t *sql.AlertV2Operand) *materializedOperand {
	if t == nil {
		return nil
	}
	out := &materializedOperand{Column: t.Column}
	if v := t.Value; v != nil {
		out.Value = &materializedOperandValue{
			BoolValue:   v.BoolValue,
			StringValue: v.StringValue,
		}
		if v.DoubleValue != 0 || slices.Contains(v.ForceSendFields, "DoubleValue") {
			d := jsonFloat(v.DoubleValue)
			out.Value.DoubleValue = &d
		}
	}
	return out
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

	notification := alert.Evaluation.Notification
	if notification != nil {
		// The backend always serializes notify_on_ok in the file, even when
		// false; the SDK marshaler would otherwise drop the zero value.
		n := *notification
		n.ForceSendFields = append(n.ForceSendFields, "NotifyOnOk")
		notification = &n
	} else {
		// The backend always writes a notification object, materializing an
		// empty {} when the alert has no notification configured.
		notification = &sql.AlertV2Notification{}
	}

	af := alertFile{
		CustomSummary: alert.CustomSummary,
		Evaluation: &materializedEvaluation{
			Source:             alert.Evaluation.Source,
			ComparisonOperator: alert.Evaluation.ComparisonOperator,
			Threshold:          materializeThreshold(alert.Evaluation.Threshold),
			Notification:       notification,
		},
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
			ObjectId:   nextTreenodeID(),
		},
		Data: data,
	}
	return nil
}

// alertNotFound mirrors the cloud's 404 for an alert that is not in the
// workspace: a purged alert is reported by its tree node, an id that never
// existed gets a per-method message.
func (s *FakeWorkspace) alertNotFound(alertId, unknownMessage string) Response {
	message := unknownMessage
	if s.PurgedAlertIDs[alertId] {
		message = fmt.Sprintf("Tree node with id %s does not exist", alertId)
	}
	return Response{
		StatusCode: http.StatusNotFound,
		Body: map[string]string{
			"error_code": "NOT_FOUND",
			"message":    message,
		},
	}
}

func (s *FakeWorkspace) AlertsGet(alertId string) Response {
	defer s.LockUnlock()()

	// A trashed (not purged) alert is still readable, with lifecycle_state DELETED.
	alert, ok := s.Alerts[alertId]
	if !ok {
		return s.alertNotFound(alertId, fmt.Sprintf("Alert with id %s not found or has no folder node internal name", alertId))
	}

	return Response{
		Body: alert,
	}
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
		existing, ok := s.Alerts[alertId]
		if !ok {
			return s.alertNotFound(alertId, fmt.Sprintf("Alert with id %s not found or has no folder node internal name", alertId))
		}
		if existing.LifecycleState == sql.AlertLifecycleStateDeleted {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body: map[string]string{
					"error_code": "INVALID_PARAMETER_VALUE",
					"message":    fmt.Sprintf("Cannot move node '%s' (ID: %s) because it is in trash folder", path.Base(alertFilePath(existing)), alertId),
				},
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
		return s.alertNotFound(alertId, fmt.Sprintf("Alert with id %s not found", alertId))
	}
	if alert.LifecycleState == sql.AlertLifecycleStateDeleted && !purge {
		return Response{
			StatusCode: http.StatusBadRequest,
			Body: map[string]string{
				"error_code": "BAD_REQUEST",
				"message":    fmt.Sprintf("Alert %s is already trashed", alertId),
			},
		}
	}

	delete(s.files, alertFilePath(alert))

	if purge {
		delete(s.Alerts, alertId)
		s.PurgedAlertIDs[alertId] = true
	} else {
		alert.LifecycleState = sql.AlertLifecycleStateDeleted
		s.Alerts[alertId] = alert
	}

	return Response{
		StatusCode: 200,
	}
}
