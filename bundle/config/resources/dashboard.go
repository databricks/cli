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

	// Direct deployment uses ForceSendFields to serialize zero values in the bundle configuration.
	// This struct [DashboardConfig] is the config representation of a dashboard. So it's
	// necessary to override the ForceSendFields from the [dashboards.Dashboard] struct here.
	//
	// This is necessary to serialize the zero value of EmbedCredentials in the local
	ForceSendFields []string `json:"-" url:"-"`
}

type Dashboard struct {
	BaseResource
	DashboardConfig

	Permissions []DashboardPermission `json:"permissions,omitempty"`

	// FilePath points to the local `.lvdash.json` file containing the dashboard definition.
	// This is inlined into serialized_dashboard during deployment. The file_path is kept around
	// as metadata which is needed for `databricks bundle generate dashboard --resource <dashboard_key>` to work.
	// This is not part of DashboardConfig because we don't need to store this in the resource state.
	FilePath string `json:"file_path,omitempty"`
}

// These override functions are necessary to ensure that "serialized_dashboard" and "embed_credentials"
// serialized into the state for direct deployment.
func (c *DashboardConfig) UnmarshalJSON(b []byte) error {
	err := marshal.Unmarshal(b, c)

	// Do not  de-serialized the nested "serialized_dashboard" field. By default, using a json decoder
	// the value from "serialized_dashboard" will be de-serialized into both
	// "DashboardConfig.Dashboard.SerializedDashboard" and "DashboardConfig.SerializedDashboard".
	// We only want to de-serialize it into "DashboardConfig.SerializedDashboard"
	// persistent diffs in direct deployment.
	embeddedDashboard := c.Dashboard
	embeddedDashboard.SerializedDashboard = ""
	forceSendFields := []string{}
	for _, field := range forceSendFields {
		if field != "SerializedDashboard" {
			forceSendFields = append(forceSendFields, field)
		}
	}
	embeddedDashboard.ForceSendFields = forceSendFields
	c.Dashboard = embeddedDashboard
	return err
}

func (c DashboardConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
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
