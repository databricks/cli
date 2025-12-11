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
	// The timestamp of when the dashboard was created.
	CreateTime string `json:"create_time,omitempty"`
	// UUID identifying the dashboard.
	DashboardId string `json:"dashboard_id,omitempty"`
	// The display name of the dashboard.
	DisplayName string `json:"display_name,omitempty"`
	// The etag for the dashboard. Can be optionally provided on updates to
	// ensure that the dashboard has not been modified since the last read. This
	// field is excluded in List Dashboards responses.
	Etag string `json:"etag,omitempty"`
	// The state of the dashboard resource. Used for tracking trashed status.
	LifecycleState dashboards.LifecycleState `json:"lifecycle_state,omitempty"`
	// The workspace path of the folder containing the dashboard. Includes
	// leading slash and no trailing slash. This field is excluded in List
	// Dashboards responses.
	ParentPath string `json:"parent_path,omitempty"`
	// The workspace path of the dashboard asset, including the file name.
	// Exported dashboards always have the file extension `.lvdash.json`. This
	// field is excluded in List Dashboards responses.
	Path string `json:"path,omitempty"`
	// The timestamp of when the dashboard was last updated by the user. This
	// field is excluded in List Dashboards responses.
	UpdateTime string `json:"update_time,omitempty"`
	// The warehouse ID used to run the dashboard.
	WarehouseId string `json:"warehouse_id,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`

	// ==============================================
	// === overrides over [dashboards.Dashboard] ===
	// ==============================================

	// SerializedDashboard holds the contents of the dashboard in serialized JSON form.
	// Even though the SDK represents this as a string, we override it as any to allow for inlining as YAML.
	// If the value is a string, it is used as is.
	// If it is not a string, its contents is marshalled as JSON.
	SerializedDashboard any `json:"serialized_dashboard,omitempty"`

	// EmbedCredentials is a flag to indicate if the publisher's credentials should
	// be embedded in the published dashboard. These embedded credentials will be used
	// to execute the published dashboard's queries.
	//
	// Defaults to false if not set.
	EmbedCredentials bool `json:"embed_credentials,omitempty"`

	// DatasetCatalog sets the default catalog for all datasets in this dashboard.
	// When set, this overrides the catalog specified in individual dataset definitions.
	DatasetCatalog string `json:"dataset_catalog,omitempty"`

	// DatasetSchema sets the default schema for all datasets in this dashboard.
	// When set, this overrides the schema specified in individual dataset definitions.
	DatasetSchema string `json:"dataset_schema,omitempty"`
}

func (c *DashboardConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c DashboardConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
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
