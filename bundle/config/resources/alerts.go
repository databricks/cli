package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type Alert struct {
	BaseResource
	sql.AlertV2 //nolint AlertV2 also defines Id and URL field with the same json tag "id" and "url"

	Permissions []AlertPermission `json:"permissions,omitempty"`

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
	baseURL.Path = "sql/alerts-v2/" + a.ID
	a.URL = baseURL.String()
}

func (a *Alert) GetName() string {
	return a.DisplayName
}

func (a *Alert) GetURL() string {
	return a.URL
}
