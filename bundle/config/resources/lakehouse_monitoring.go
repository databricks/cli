package resources

import (
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type LakehouseMonitor struct {
	// Represents the Input Arguments for Terraform and will get
	// converted to a HCL representation for CRUD
	*catalog.CreateMonitor

	// This represents the id which is the full name of the monitor
	// (catalog_name.schema_name.table_name) that can be used
	// as a reference in other resources. This value is returned by terraform.
	ID string `json:"id,omitempty" bundle:"readonly"`

	// Path to config file where the resource is defined. All bundle resources
	// include this for interpolation purposes.
	paths.Paths

	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
}
