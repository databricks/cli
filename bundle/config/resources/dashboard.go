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

	// DisplayName is the name of the dashboard (both as title and as basename in the workspace).
	DisplayName string `json:"display_name,omitempty"`

	// ParentPath is the path to the parent directory of the dashboard.
	ParentPath string `json:"parent_path,omitempty"`

	// WarehouseID is the ID of the warehouse to use for the dashboard.
	WarehouseID string `json:"warehouse_id,omitempty"`

	// ===========================
	// ==== END OF API FIELDS ====
	// ===========================

	// DefinitionPath points to the local `.lvdash.json` file containing the dashboard definition.
	DefinitionPath string `json:"definition_path,omitempty"`
}

func (s *Dashboard) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Dashboard) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

func (_ *Dashboard) Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error) {
	_, err := w.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
		DashboardId: id,
	})
	if err != nil {
		log.Debugf(ctx, "Dashboard %s does not exist", id)
		return false, err
	}
	return true, nil
}

func (_ *Dashboard) TerraformResourceName() string {
	return "databricks_dashboard"
}
