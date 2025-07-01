package resources

import (
	"context"
	"fmt"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

type DashboardPermissionLevel string

// DashboardPermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any dashboard.
type DashboardPermission struct {
	Level DashboardPermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type DashboardConfig struct {
	dashboards.Dashboard

	// =========================
	// === Additional fields ===
	// =========================

	// SerializedDashboard holds the contents of the dashboard in serialized JSON form.
	// We override the field's type from the SDK struct here to allow for inlining as YAML.
	// If the value is a string, it is used as is.
	// If it is not a string, its contents is marshalled as JSON.
	SerializedDashboard any `json:"serialized_dashboard,omitempty"`

	// EmbedCredentials is a flag to indicate if the publisher's credentials should
	// be embedded in the published dashboard. These embedded credentials will be used
	// to execute the published dashboard's queries.
	//
	// Defaults to false if not set.
	EmbedCredentials bool `json:"embed_credentials,omitempty"`
}

type Dashboard struct {
	ID             string                `json:"id,omitempty" bundle:"readonly"`
	Permissions    []DashboardPermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus        `json:"modified_status,omitempty" bundle:"internal"`
	URL            string                `json:"url,omitempty" bundle:"internal"`

	DashboardConfig

	// FilePath points to the local `.lvdash.json` file containing the dashboard definition.
	// This is inlined into serialized_dashboard during deployment. The file_path is kept around
	// as metadata which is needed for `databricks bundle generate dashboard --resource <dashboard_key>` to work.
	// This is not part of DashboardConfig because we don't need to store this in the resource state.
	FilePath string `json:"file_path,omitempty"`
}

func (r *Dashboard) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r Dashboard) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

func (*Dashboard) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
		DashboardId: id,
	})
	if err != nil {
		log.Debugf(ctx, "dashboard %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*Dashboard) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "dashboard",
		PluralName:    "dashboards",
		SingularTitle: "Dashboard",
		PluralTitle:   "Dashboards",
	}
}

func (r *Dashboard) InitializeURL(baseURL url.URL) {
	if r.ID == "" {
		return
	}

	baseURL.Path = fmt.Sprintf("dashboardsv3/%s/published", r.ID)
	r.URL = baseURL.String()
}

func (r *Dashboard) GetName() string {
	return r.DisplayName
}

func (r *Dashboard) GetURL() string {
	return r.URL
}
