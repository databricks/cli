package testserver

import (
	"encoding/json"
	"fmt"
	alerts "github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/utils"
	workspaceapi "github.com/databricks/cli/libs/workspaceapi"
	"net/http"
	"path"
	"strconv"
	"strings"
)

// alertFile mirrors the schema of a .dbalert.json file. It is the inverse of
// bundle/config/mutator/load_dbalert_files.go: API-only fields (display_name,
// warehouse_id) are dropped and the joined query_text/custom_description are
// split back into lines. The field order matches what the backend materializes.
type alertFile struct {
	CustomSummary          string                  `json:"custom_summary,omitempty"`
	Evaluation             *materializedEvaluation `json:"evaluation,omitempty"`
	Schedule               *alerts.CronSchedule    `json:"schedule,omitempty"`
	QueryLines             []string                `json:"query_lines,omitempty"`
	CustomDescriptionLines []string                `json:"custom_description_lines,omitempty"`
}

// materializedEvaluation mirrors the field order the alerts backend uses when it
// writes the .dbalert.json file, which differs from alerts.AlertEvaluation: the
// backend emits source before comparison_operator and always writes a
// notification object (an empty {} when none is configured).
type materializedEvaluation struct {
	Source             alerts.AlertOperandColumn `json:"source"`
	ComparisonOperator alerts.ComparisonOperator `json:"comparison_operator"`
	Threshold          *materializedOperand      `json:"threshold,omitempty"`
	Notification       *alerts.AlertNotification `json:"notification"`
}

// materializedOperand and materializedOperandValue mirror alerts.AlertOperand but
// format double_value via jsonFloat so it keeps its decimal point in the file.
type materializedOperand struct {
	Column *alerts.AlertOperandColumn `json:"column,omitempty"`
	Value  *materializedOperandValue  `json:"value,omitempty"`
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
// double_value is emitted whenever the source request set it, even when it is
// the zero value.
func materializeThreshold(t *alerts.AlertOperand) *materializedOperand {
	if t == nil {
		return nil
	}
	out := &materializedOperand{Column: t.Column}
	if v := t.Value; v != nil {
		out.Value = &materializedOperandValue{
			BoolValue:   utils.Deref(v.BoolValue),
			StringValue: utils.Deref(v.StringValue),
		}
		if v.DoubleValue != nil {
			d := jsonFloat(*v.DoubleValue)
			out.Value.DoubleValue = &d
		}
	}
	return out
}

// alertFilePath returns the workspace path of the .dbalert.json file the backend
// materializes for an alert: <parent_path>/<display_name>.dbalert.json.
func alertFilePath(alert alerts.Alert) string {
	return path.Join(utils.Deref(alert.ParentPath), utils.Deref(alert.DisplayName)+".dbalert.json")
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
func (s *FakeWorkspace) writeAlertFile(alert alerts.Alert) error {
	if utils.Deref(alert.ParentPath) == "" || utils.Deref(alert.DisplayName) == "" {
		return nil
	}

	evaluation := alert.Evaluation
	if evaluation == nil {
		evaluation = &alerts.AlertEvaluation{}
	}
	notification := evaluation.Notification
	if notification != nil {
		// The backend always serializes notify_on_ok in the file, even when false.
		n := *notification
		n.NotifyOnOk = new(utils.Deref(n.NotifyOnOk))
		notification = &n
	} else {
		// The backend always writes a notification object, materializing an
		// empty {} when the alert has no notification configured.
		notification = &alerts.AlertNotification{}
	}

	af := alertFile{
		CustomSummary: utils.Deref(alert.CustomSummary),
		Evaluation: &materializedEvaluation{
			Source:             utils.Deref(evaluation.Source),
			ComparisonOperator: evaluation.ComparisonOperator,
			Threshold:          materializeThreshold(evaluation.Threshold),
			Notification:       notification,
		},
		Schedule:               alert.Schedule,
		QueryLines:             splitLines(utils.Deref(alert.QueryText)),
		CustomDescriptionLines: splitLines(utils.Deref(alert.CustomDescription)),
	}

	data, err := json.MarshalIndent(af, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	filePath := alertFilePath(alert)
	s.files[filePath] = FileEntry{
		Info: workspaceapi.ObjectInfo{
			ObjectType: "FILE",
			Path:       filePath,
			ObjectId:   nextTreenodeID(),
		},
		Data: data,
	}
	return nil
}

func (s *FakeWorkspace) AlertsUpsert(req Request, alertId string) Response {
	var alert alerts.Alert

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

	alert.Id = utils.NonZero(alertId)
	alert.LifecycleState = alerts.AlertLifecycleState_Active
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
		alert.LifecycleState = alerts.AlertLifecycleState_Deleted
		s.Alerts[alertId] = alert
	}

	return Response{
		StatusCode: 200,
	}
}
