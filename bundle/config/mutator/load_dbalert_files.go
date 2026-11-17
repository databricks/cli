package mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type loadDBAlertFiles struct{}

func LoadDBAlertFiles() bundle.Mutator {
	return &loadDBAlertFiles{}
}

func (m *loadDBAlertFiles) Name() string {
	return "LoadDBAlertFiles"
}

type AlertFile struct {
	sql.AlertV2

	// query_text and custom_description can be split into lines to make it easier to view the diff
	// in a Git editor.
	QueryLines             []string `json:"query_lines,omitempty"`
	CustomDescriptionLines []string `json:"custom_description_lines,omitempty"`
}

func (d *AlertFile) UnmarshalJSON(data []byte) error {
	return marshal.Unmarshal(data, d)
}

func (d AlertFile) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(d)
}

func (m *loadDBAlertFiles) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Fields that are only settable in the API, and are not allowed in .dbalert.json.
	// We will only allow these fields to be set in the bundle YAML when an .dbalert.json is
	// specified. This is done to only have one way to set these fields when a .dbalert.json is
	// specified.
	allowedInYAML := []string{"warehouse_id", "display_name", "file_path", "permissions", "lifecycle"}

	for alertKey, alert := range b.Config.Resources.Alerts {
		if alert.FilePath == "" {
			continue
		}

		alertV, err := dyn.GetByPath(b.Config.Value(), dyn.NewPath(dyn.Key("resources"), dyn.Key("alerts"), dyn.Key(alertKey)))
		if err != nil {
			return diag.FromErr(err)
		}

		if isAlertQueryFile(alert.FilePath) {
			if diags := loadAlertQueryFile(b, alertKey, alert, alertV); diags.HasError() {
				return diags
			}
			continue
		}

		// No other fields other than allowedInYAML should be set in the bundle YAML.
		m, ok := alertV.AsMap()
		if !ok {
			return diag.FromErr(fmt.Errorf("internal error: alert value is not a map, got %s", alertV.Kind()))
		}

		for _, p := range m.Pairs() {
			k := p.Key.MustString()
			v := p.Value

			if slices.Contains(allowedInYAML, k) {
				continue
			}

			if v.Kind() == dyn.KindNil || v.Kind() == dyn.KindInvalid {
				continue
			}

			return diag.Diagnostics{
				{
					ID:        "",
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("field %s is not allowed in the bundle configuration.", k),
					Detail:    "When a .dbalert.json is specified, only the following fields are allowed in the bundle configuration: " + strings.Join(allowedInYAML, ", "),
					Paths:     structpath.NewPathSlice("resources", "alerts", alertKey, k),
					Locations: v.Locations(),
				},
			}
		}

		content, err := os.ReadFile(alertFilePath(b, alert.FilePath))
		if err != nil {
			return diag.Diagnostics{
				{
					ID:        diag.ID(""),
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("failed to read .dbalert.json file %s: %s", alert.FilePath, err),
					Detail:    "",
					Paths:     structpath.NewPathSlice("resources", "alerts", alertKey, "file_path"),
					Locations: alertV.Get("file_path").Locations(),
				},
			}
		}

		var dbalertFromFile AlertFile
		err = json.Unmarshal(content, &dbalertFromFile)
		if err != nil {
			return diag.Diagnostics{
				{
					ID:        diag.ID(""),
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("failed to parse .dbalert.json file %s: %s", alert.FilePath, err),
					Detail:    "",
					Paths:     structpath.NewPathSlice("resources", "alerts", alertKey, "file_path"),
					Locations: alertV.Get("file_path").Locations(),
				},
			}
		}

		// Check that the file does not have any variable interpolations.
		if dynvar.ContainsVariableReference(string(content)) {
			return diag.Diagnostics{
				{
					ID:        diag.ID(""),
					Severity:  diag.Error,
					Summary:   fmt.Sprintf(".alert file %s must not contain variable interpolations.", alert.FilePath),
					Detail:    "Please inline the alert configuration in the bundle configuration to use variables",
					Paths:     structpath.NewPathSlice("resources", "alerts", alertKey, "file_path"),
					Locations: alertV.Get("file_path").Locations(),
				},
			}
		}

		var queryText string
		if len(dbalertFromFile.QueryLines) > 0 {
			var sb strings.Builder
			for _, line := range dbalertFromFile.QueryLines {
				sb.WriteString(line)
				sb.WriteString("\n")
			}
			queryText = sb.String()
		}

		var customDescription string
		if len(dbalertFromFile.CustomDescriptionLines) > 0 {
			var sb strings.Builder
			for _, line := range dbalertFromFile.CustomDescriptionLines {
				sb.WriteString(line)
				sb.WriteString("\n")
			}
			customDescription = sb.String()
		}

		newAlert := sql.AlertV2{
			// Fields with different schema in file vs API.
			CustomDescription: customDescription,
			QueryText:         queryText,

			// API only fields. All these should be present in [allowedInYAML]
			DisplayName: alert.DisplayName,
			WarehouseId: alert.WarehouseId,

			// Fields with the same schema in file vs API.
			CustomSummary:        dbalertFromFile.CustomSummary,
			Schedule:             dbalertFromFile.Schedule,
			Evaluation:           dbalertFromFile.Evaluation,
			EffectiveRunAs:       dbalertFromFile.EffectiveRunAs,
			RunAs:                dbalertFromFile.RunAs,
			RunAsUserName:        dbalertFromFile.RunAsUserName, //nolint:staticcheck // SA1019: run_as_user_name is still accepted in alert files; run_as is the replacement
			ParentPath:           dbalertFromFile.ParentPath,
			Parameters:           dbalertFromFile.Parameters,
			CustomTemplateFormat: dbalertFromFile.CustomTemplateFormat,

			// Output only fields.
			CreateTime:     dbalertFromFile.CreateTime,
			OwnerUserName:  dbalertFromFile.OwnerUserName,
			UpdateTime:     dbalertFromFile.UpdateTime,
			LifecycleState: dbalertFromFile.LifecycleState,

			// Other fields.
			Id:              dbalertFromFile.Id,
			ForceSendFields: dbalertFromFile.ForceSendFields,
		}

		// write values from the file to the alert.
		b.Config.Resources.Alerts[alertKey].AlertV2 = newAlert
	}

	return nil
}

// alertFilePath anchors a file_path to the bundle root. NormalizePaths rewrote
// file_path to be relative to the bundle root, but the process working directory
// can be any directory within the bundle. Absolute paths are kept as-is by
// NormalizePaths.
func alertFilePath(b *bundle.Bundle, filePath string) string {
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(b.BundleRootPath, filePath)
}

// isAlertQueryFile reports whether file_path points to a .sql file holding only
// the alert query, as opposed to a full .dbalert.json alert definition.
func isAlertQueryFile(filePath string) bool {
	return strings.EqualFold(filepath.Ext(filePath), ".sql")
}

// loadAlertQueryFile sets query_text from a .sql file. Unlike a .dbalert.json, the
// file only holds the query, so all other alert fields stay in the bundle YAML.
func loadAlertQueryFile(b *bundle.Bundle, alertKey string, alert *resources.Alert, alertV dyn.Value) diag.Diagnostics {
	filePathV := alertV.Get("file_path")
	filePathPaths := structpath.NewPathSlice("resources", "alerts", alertKey, "file_path")

	if q := alertV.Get("query_text"); q.Kind() != dyn.KindInvalid && q.Kind() != dyn.KindNil {
		return diag.Diagnostics{
			{
				ID:        "",
				Severity:  diag.Error,
				Summary:   "query_text cannot be set when file_path points to a .sql file",
				Detail:    fmt.Sprintf("The alert query is read from %s. Remove either query_text or file_path.", alert.FilePath),
				Paths:     structpath.NewPathSlice("resources", "alerts", alertKey, "query_text"),
				Locations: q.Locations(),
			},
		}
	}

	content, err := os.ReadFile(alertFilePath(b, alert.FilePath))
	if err != nil {
		return diag.Diagnostics{
			{
				ID:        "",
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("failed to read alert query file %s: %s", alert.FilePath, err),
				Detail:    "",
				Paths:     filePathPaths,
				Locations: filePathV.Locations(),
			},
		}
	}

	query := string(content)
	if strings.TrimSpace(query) == "" {
		return diag.Diagnostics{
			{
				ID:        "",
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("alert query file %s is empty", alert.FilePath),
				Detail:    "",
				Paths:     filePathPaths,
				Locations: filePathV.Locations(),
			},
		}
	}

	// The file is read after variables are resolved, so references in it would be
	// sent to the backend verbatim. Reject them, like for .dbalert.json files.
	if dynvar.ContainsVariableReference(query) {
		return diag.Diagnostics{
			{
				ID:        "",
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("alert query file %s must not contain variable interpolations.", alert.FilePath),
				Detail:    "Please set query_text inline in the bundle configuration to use variables",
				Paths:     filePathPaths,
				Locations: filePathV.Locations(),
			},
		}
	}

	alert.QueryText = query
	return nil
}
