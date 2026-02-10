package resources

import (
	"context"
	"net/url"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"

	"github.com/databricks/cli/libs/log"
)

type ExternalLocationGrantPrivilege string

const (
	ExternalLocationGrantPrivilegeAllPrivileges        ExternalLocationGrantPrivilege = "ALL_PRIVILEGES"
	ExternalLocationGrantPrivilegeCreateExternalTable  ExternalLocationGrantPrivilege = "CREATE_EXTERNAL_TABLE"
	ExternalLocationGrantPrivilegeCreateExternalVolume ExternalLocationGrantPrivilege = "CREATE_EXTERNAL_VOLUME"
	ExternalLocationGrantPrivilegeCreateManagedStorage ExternalLocationGrantPrivilege = "CREATE_MANAGED_STORAGE"
	ExternalLocationGrantPrivilegeCreateTable          ExternalLocationGrantPrivilege = "CREATE_TABLE"
	ExternalLocationGrantPrivilegeCreateVolume         ExternalLocationGrantPrivilege = "CREATE_VOLUME"
	ExternalLocationGrantPrivilegeManage               ExternalLocationGrantPrivilege = "MANAGE"
	ExternalLocationGrantPrivilegeReadFiles            ExternalLocationGrantPrivilege = "READ_FILES"
	ExternalLocationGrantPrivilegeWriteFiles           ExternalLocationGrantPrivilege = "WRITE_FILES"
)

// Values returns all valid ExternalLocationGrantPrivilege values
func (ExternalLocationGrantPrivilege) Values() []ExternalLocationGrantPrivilege {
	return []ExternalLocationGrantPrivilege{
		ExternalLocationGrantPrivilegeAllPrivileges,
		ExternalLocationGrantPrivilegeCreateExternalTable,
		ExternalLocationGrantPrivilegeCreateExternalVolume,
		ExternalLocationGrantPrivilegeCreateManagedStorage,
		ExternalLocationGrantPrivilegeCreateTable,
		ExternalLocationGrantPrivilegeCreateVolume,
		ExternalLocationGrantPrivilegeManage,
		ExternalLocationGrantPrivilegeReadFiles,
		ExternalLocationGrantPrivilegeWriteFiles,
	}
}

// ExternalLocationGrant holds the grant level settings for a single principal in Unity Catalog.
// Multiple of these can be defined on any external location.
type ExternalLocationGrant struct {
	Privileges []ExternalLocationGrantPrivilege `json:"privileges"`

	Principal string `json:"principal"`
}

type ExternalLocation struct {
	// Manually include BaseResource fields to avoid URL field conflict
	ID             string         `json:"id,omitempty" bundle:"readonly"`
	ModifiedStatus ModifiedStatus `json:"modified_status,omitempty" bundle:"internal"`
	// Note: We intentionally don't include BaseResource.URL here to avoid conflict with Url field below
	Lifecycle Lifecycle `json:"lifecycle,omitempty"`

	catalog.CreateExternalLocation

	// List of grants to apply on this external location.
	Grants []ExternalLocationGrant `json:"grants,omitempty"`
}

func (e *ExternalLocation) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.ExternalLocations.GetByName(ctx, name)
	if err != nil {
		log.Debugf(ctx, "external location with name %s does not exist: %v", name, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*ExternalLocation) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "external_location",
		PluralName:    "external_locations",
		SingularTitle: "External Location",
		PluralTitle:   "External Locations",
	}
}

func (e *ExternalLocation) InitializeURL(baseURL url.URL) {
	// External locations don't have a workspace URL
	// The Url field is for the storage path (s3://...), not a workspace URL
}

func (e *ExternalLocation) GetURL() string {
	// Return empty as external locations don't have a workspace URL
	return ""
}

func (e *ExternalLocation) GetName() string {
	return e.Name
}

func (e *ExternalLocation) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, e)
}

func (e ExternalLocation) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(e)
}
