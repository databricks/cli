package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

type SqlWarehousePermissionLevel string

type SqlWarehouse struct {
	ID string `json:"id,omitempty" bundle:"readonly"`

	Permissions    []SqlWarehousePermissionLevel `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus                `json:"modified_status,omitempty" bundle:"internal"`
	URL            string                        `json:"url,omitempty" bundle:"internal"`

	sql.CreateWarehouseRequest
}

func (w *SqlWarehouse) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, w)
}

func (w SqlWarehouse) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(w)
}

func (sw *SqlWarehouse) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Warehouses.GetByName(ctx, name)
	if err != nil {
		log.Debugf(ctx, "sql warehouse %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (*SqlWarehouse) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "sql_warehouse",
		PluralName:    "sql_warehouses",
		SingularTitle: "Sql Warehouse",
		PluralTitle:   "Sql Warehouses",
	}
}

func (sw *SqlWarehouse) InitializeURL(baseURL url.URL) {
	if sw.ModifiedStatus == "" || sw.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	baseURL.Path = "apps/" + sw.GetName()
	sw.URL = baseURL.String()
}

func (sw *SqlWarehouse) GetName() string {
	return sw.Name
}

func (sw *SqlWarehouse) GetURL() string {
	return sw.URL
}
