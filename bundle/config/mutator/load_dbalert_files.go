package mutator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type loadDBAlertFiles struct{}

func LoadDBAlertFiles() bundle.Mutator {
	return &loadDBAlertFiles{}
}

func (m *loadDBAlertFiles) Name() string {
	return "LoadDBAlertFiles"
}

type dbalertFile struct {
	sql.AlertV2

	// query_text and custom_description are split into lines to make it easier to view the diff
	// in a Git editor.
	QueryLines             []string `json:"query_lines,omitempty"`
	CustomDescriptionLines []string `json:"custom_description_lines,omitempty"`
}

func (m *loadDBAlertFiles) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Fields that are only settable in the API, and are not allowed in .dbalert.json.
	// We will only allow these fields to be set in the bundle YAML when an .dbalert.json is
	// specified. This is done to only have one way to set these fields when a .dbalert.json is
	// specified.
	allowedInYAML := []string{"warehouse_id", "display_name", "file_path"}

	for k, alert := range b.Config.Resources.Alerts {
		if alert.FilePath == "" {
			continue
		}

		alertV, err := dyn.GetByPath(b.Config.Value(), dyn.NewPath(dyn.Key("resources"), dyn.Key("alerts"), dyn.Key(k)))
		if err != nil {
			return diag.FromErr(err)
		}

		// No other fields other than allowedInYAML should be set in the bundle YAML.
		m, ok := alertV.AsMap()
		if !ok {
			return diag.FromErr(fmt.Errorf("internal error: alert value is not a map: %w", err))
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
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("field %s is not allowed in the bundle YAML when a .dbalert.json is specified. Please set it in the .dbalert.json file instead. Only allowed fields are: %s", k, strings.Join(allowedInYAML, ", ")),
					Paths:     []dyn.Path{dyn.MustPathFromString(fmt.Sprintf("resources.alerts.%s.%s", k, k))},
					Locations: v.Locations(),
				},
			}
		}

		content, err := os.ReadFile(alert.FilePath)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("failed to read .dbalert.json file %s: %s", alert.FilePath, err),
					Paths:     []dyn.Path{dyn.MustPathFromString(fmt.Sprintf("resources.alerts.%s.file_path", k))},
					Locations: alertV.Get("file_path").Locations(),
				},
			}
		}

		var dbalertFromFile dbalertFile
		err = json.Unmarshal(content, &dbalertFromFile)
		if err != nil {
			return diag.Diagnostics{
				{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf("failed to parse .dbalert.json file %s: %s", alert.FilePath, err),
					Paths:     []dyn.Path{dyn.MustPathFromString(fmt.Sprintf("resources.alerts.%s.file_path", k))},
					Locations: alertV.Get("file_path").Locations(),
				},
			}
		}

		// Check that the file does not have any variable interpolations.
		if dynvar.ContainsVariableReference(string(content)) {
			return diag.Diagnostics{
				{
					Severity:  diag.Error,
					Summary:   fmt.Sprintf(".dbalert.json file %s contains variable interpolations, which are not supported. Please inline the alert configuration in the bundle YAML to use variables", alert.FilePath),
					Paths:     []dyn.Path{dyn.MustPathFromString(fmt.Sprintf("resources.alerts.%s.file_path", k))},
					Locations: alertV.Get("file_path").Locations(),
				},
			}
		}

		// Handle query_text: prefer query_lines if present, otherwise use query_text directly.
		queryText := ""
		if len(dbalertFromFile.QueryLines) > 0 {
			for _, line := range dbalertFromFile.QueryLines {
				queryText += line + "\n"
			}
		} else {
			queryText = dbalertFromFile.QueryText
		}

		// Handle custom_description: prefer custom_description_lines if present, otherwise use custom_description directly.
		customDescription := ""
		if len(dbalertFromFile.CustomDescriptionLines) > 0 {
			for _, line := range dbalertFromFile.CustomDescriptionLines {
				customDescription += line + "\n"
			}
		} else {
			customDescription = dbalertFromFile.CustomDescription
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
		b.Config.Resources.Alerts[k].AlertV2 = newAlert
	}

	return nil
}
