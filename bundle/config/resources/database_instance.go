package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/workspaceurls"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type DatabaseInstance struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	database.DatabaseInstance

	Permissions []Permission `json:"permissions,omitempty"`
}

func (d *DatabaseInstance) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, d)
}

func (d DatabaseInstance) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(d)
}

func (d *DatabaseInstance) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "database instance %s does not exist", name)
		return false, err
	}
	return true, nil
}

var databaseInstanceDescription = ResourceDescription{
	SingularName:  "database_instance",
	PluralName:    "database_instances",
	SingularTitle: "Database instance",
	PluralTitle:   "Database instances",
	Permissions: PermissionsDescription{
		BundleLevels: BundlePermissionLevels{
			CanManage: "CAN_MANAGE",
			CanView:   "CAN_USE",
		},
		RequestObjectType:  "database-instances",
		ResponseObjectType: "database-instances",
		ObjectIDField:      "id",
	},
}

func (d *DatabaseInstance) ResourceDescription() ResourceDescription {
	return databaseInstanceDescription
}

func (d *DatabaseInstance) GetName() string {
	return d.Name
}

func (d *DatabaseInstance) InitializeURL(baseURL url.URL) {
	if d.ModifiedStatus == ModifiedStatusCreated {
		return
	}
	if d.Name == "" {
		return
	}
	d.URL = workspaceurls.ResourceURL(baseURL, "database_instances", d.Name)
}
