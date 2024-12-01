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

type Dashboard struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	URL            string         `json:"url,omitempty" bundle:"internal"`

	*dashboards.Dashboard

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

	// FilePath points to the local `.lvdash.json` file containing the dashboard definition.
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

func (*Dashboard) TerraformResourceName() string {
	return "databricks_dashboard"
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

func (r *Dashboard) IsNil() bool {
	return r.Dashboard == nil
}
