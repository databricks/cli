package resources

import (
	"context"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

type Dashboard struct {
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	Permissions    []Permission   `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`

	// ===========================
	// === BEGIN OF API FIELDS ===
	// ===========================

	// DisplayName is the display name of the dashboard (both as title and as basename in the workspace).
	DisplayName string `json:"display_name"`

	// WarehouseID is the ID of the SQL Warehouse used to run the dashboard's queries.
	WarehouseID string `json:"warehouse_id"`

	// SerializedDashboard holds the contents of the dashboard in serialized JSON form.
	// Note: its type is any and not string such that it can be inlined as YAML.
	// If it is not a string, its contents will be marshalled as JSON.
	SerializedDashboard any `json:"serialized_dashboard,omitempty"`

	// ParentPath is the workspace path of the folder containing the dashboard.
	// Includes leading slash and no trailing slash.
	//
	// Defaults to ${workspace.resource_path} if not set.
	ParentPath string `json:"parent_path,omitempty"`

	// EmbedCredentials is a flag to indicate if the publisher's credentials should
	// be embedded in the published dashboard. These embedded credentials will be used
	// to execute the published dashboard's queries.
	//
	// Defaults to false if not set.
	EmbedCredentials bool `json:"embed_credentials,omitempty"`

	// ===========================
	// ==== END OF API FIELDS ====
	// ===========================

	// FilePath points to the local `.lvdash.json` file containing the dashboard definition.
	FilePath string `json:"file_path,omitempty"`
}

func (s *Dashboard) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Dashboard) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (*Dashboard) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
		DashboardId: id,
	})
	if err != nil {
		log.Debugf(ctx, "Dashboard %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (*Dashboard) TerraformResourceName() string {
	return "databricks_dashboard"
}
