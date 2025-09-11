package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type AlertsPermissionLevel string

func (l AlertsPermissionLevel) Values() []string {
	return []string{
		"CAN_EDIT",
		"CAN_MANAGE",
		"CAN_READ",
		"CAN_RUN",
	}
}

type AlertPermission struct {
	Level string `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type Alert struct {
	BaseResource
	sql.AlertV2 //nolint AlertV2 also defines Id and URL field with the same json tag "id" and "url"

	Permissions []AlertPermission `json:"permissions,omitempty"`
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
