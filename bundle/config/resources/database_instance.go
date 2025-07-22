package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type DatabaseInstancePermissionLevel string

// DatabaseInstancePermission holds the permission level setting for a single principal.
// Multiple of these can be defined on any database instance.
type DatabaseInstancePermission struct {
	Level DatabaseInstancePermissionLevel `json:"level"`

	UserName             string `json:"user_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
}

type DatabaseInstance struct {
	ID             string                       `json:"id,omitempty" bundle:"readonly"`
	URL            string                       `json:"url,omitempty" bundle:"internal"`
	Permissions    []DatabaseInstancePermission `json:"permissions,omitempty"`
	ModifiedStatus ModifiedStatus               `json:"modified_status,omitempty" bundle:"internal"`

	database.DatabaseInstance
}

func (d *DatabaseInstance) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Database.GetDatabaseInstance(ctx, database.GetDatabaseInstanceRequest{Name: name})
	if err != nil {
		log.Debugf(ctx, "database instance %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (d *DatabaseInstance) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "database_instance",
		PluralName:    "database_instances",
		SingularTitle: "Database instance",
		PluralTitle:   "Database instances",
	}
}

func (d *DatabaseInstance) GetName() string {
	return d.Name
}

func (d *DatabaseInstance) GetURL() string {
	return d.URL
}

func (d *DatabaseInstance) InitializeURL(baseURL url.URL) {
	if d.Name == "" {
		return
	}
	baseURL.Path = "compute/database-instances/" + d.Name
	d.URL = baseURL.String()
}
