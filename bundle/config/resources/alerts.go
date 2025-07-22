package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type AlertPermissionLevel string

type AlertPermission struct {
	Level AlertPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type Alert struct {
	ID             string            `json:"id,omitempty" bundle:"readonly"`
	Permissions    []AlertPermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus    `json:"modified_status,omitempty" bundle:"internal"`
	URL            string            `json:"url,omitempty" bundle:"internal"`

	sql.AlertV2
}

func (a *Alert) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, a)
}

func (a Alert) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(a)
}

// TODO: check this works in context where it's used.
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

// TODO: Test that "bundle open" works with this.
func (a *Alert) GetURL() string {
	return a.URL
}
