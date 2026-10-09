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
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

		alertPath := structpath.NewPath(nil, "resources", "alerts", alertKey)
		alertV := b.Config.View().Lookup(alertPath)

		// No other fields other than allowedInYAML should be set in the bundle YAML.
		if alertV.Kind() != structvar.KindMap {
			return diag.FromErr(fmt.Errorf("internal error: alert value is not a map, got %s", alertV.Kind()))
		}

		for k, v := range alertV.MapItems() {

			if slices.Contains(allowedInYAML, k) {
				continue
			}

			if v.Kind() == structvar.KindNil || v.Kind() == structvar.KindInvalid {
				continue
			}

			return diag.Diagnostics{
				{
					ID:        "",
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("field %s is not allowed in the bundle configuration.", k),
					Detail:    "When a .dbalert.json is specified, only the following fields are allowed in the bundle configuration: " + strings.Join(allowedInYAML, ", "),
					Locations: nil,
					Paths:     []*structpath.PathNode{structpath.NewStringKey(alertPath, k)},
				},
			}
		}

		// NormalizePaths rewrote file_path to be relative to the bundle root, but the
		// process working directory can be any directory within the bundle, so anchor
		// the read to the bundle root. Absolute paths are kept as-is by NormalizePaths.
		filePath := alert.FilePath
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(b.BundleRootPath, filePath)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return diag.Diagnostics{
				{
					ID:        diag.ID(""),
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("failed to read .dbalert.json file %s: %s", alert.FilePath, err),
					Detail:    "",
					Locations: nil,
					Paths:     []*structpath.PathNode{structpath.NewStringKey(alertPath, "file_path")},
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
					Locations: nil,
					Paths:     []*structpath.PathNode{structpath.NewStringKey(alertPath, "file_path")},
				},
			}
		}

		// Check that the file does not have any variable interpolations.
		if structvar.ContainsVariableReference(string(content)) {
			return diag.Diagnostics{
				{
					ID:        diag.ID(""),
					Severity:  diag.Error,
					Summary:   fmt.Sprintf(".alert file %s must not contain variable interpolations.", alert.FilePath),
					Detail:    "Please inline the alert configuration in the bundle configuration to use variables",
					Locations: nil,
					Paths:     []*structpath.PathNode{structpath.NewStringKey(alertPath, "file_path")},
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
			CustomSummary:  dbalertFromFile.CustomSummary,
			Schedule:       dbalertFromFile.Schedule,
			Evaluation:     dbalertFromFile.Evaluation,
			EffectiveRunAs: dbalertFromFile.EffectiveRunAs,
			RunAs:          dbalertFromFile.RunAs,
			RunAsUserName:  dbalertFromFile.RunAsUserName,
			ParentPath:     dbalertFromFile.ParentPath,
			Parameters:     dbalertFromFile.Parameters,

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
