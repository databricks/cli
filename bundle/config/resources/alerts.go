package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type Alert struct {
	BaseResource
	sql.AlertV2 //nolint:govet // AlertV2.Id and our depth-0 ID field both carry json:"id"; the depth-0 field wins
	// ID shadows the same-depth collision between BaseResource.ID and
	// AlertV2.Id — both embed json:"id" at depth 1.
	ID string `json:"id,omitempty" bundle:"readonly"`

	Permissions []Permission `json:"permissions,omitempty"`

	// Filepath points to the local .dbalert.json file containing the alert definition.
	// If specified, any fields that are part of the .dbalert.json file schema will not be allowed in
	// the bundle config.
	FilePath string `json:"file_path,omitempty"`
}

func (a *Alert) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, a)
}

func (a Alert) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(a)
}

func (a *Alert) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.AlertsV2.GetAlertById(ctx, id)
	if err != nil {
		log.Debugf(ctx, "alert %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (a *Alert) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "alert",
		PluralName:    "alerts",
		SingularTitle: "Alert",
		PluralTitle:   "Alerts",
	}
}

func (a *Alert) InitializeURL(baseURL url.URL) {
	if a.ID == "" {
		return
	}
	a.URL = workspaceurls.ResourceURL(baseURL, "alerts", a.ID)
}

func (a *Alert) GetName() string {
	return a.DisplayName
}
