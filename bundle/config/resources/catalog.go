package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"

	"github.com/databricks/cli/libs/log"
)

type CatalogGrantPrivilege string

const (
	CatalogGrantPrivilegeAllPrivileges     CatalogGrantPrivilege = "ALL_PRIVILEGES"
	CatalogGrantPrivilegeApplyTag          CatalogGrantPrivilege = "APPLY_TAG"
	CatalogGrantPrivilegeCreateConnection  CatalogGrantPrivilege = "CREATE_CONNECTION"
	CatalogGrantPrivilegeCreateExternalLocation CatalogGrantPrivilege = "CREATE_EXTERNAL_LOCATION"
	CatalogGrantPrivilegeCreateExternalTable    CatalogGrantPrivilege = "CREATE_EXTERNAL_TABLE"
	CatalogGrantPrivilegeCreateExternalVolume   CatalogGrantPrivilege = "CREATE_EXTERNAL_VOLUME"
	CatalogGrantPrivilegeCreateForeignCatalog   CatalogGrantPrivilege = "CREATE_FOREIGN_CATALOG"
	CatalogGrantPrivilegeCreateFunction         CatalogGrantPrivilege = "CREATE_FUNCTION"
	CatalogGrantPrivilegeCreateManagedStorage   CatalogGrantPrivilege = "CREATE_MANAGED_STORAGE"
	CatalogGrantPrivilegeCreateMaterializedView CatalogGrantPrivilege = "CREATE_MATERIALIZED_VIEW"
	CatalogGrantPrivilegeCreateModel            CatalogGrantPrivilege = "CREATE_MODEL"
	CatalogGrantPrivilegeCreateSchema           CatalogGrantPrivilege = "CREATE_SCHEMA"
	CatalogGrantPrivilegeCreateStorageCredential CatalogGrantPrivilege = "CREATE_STORAGE_CREDENTIAL"
	CatalogGrantPrivilegeCreateTable            CatalogGrantPrivilege = "CREATE_TABLE"
	CatalogGrantPrivilegeCreateVolume           CatalogGrantPrivilege = "CREATE_VOLUME"
	CatalogGrantPrivilegeExecute                CatalogGrantPrivilege = "EXECUTE"
	CatalogGrantPrivilegeManage                 CatalogGrantPrivilege = "MANAGE"
	CatalogGrantPrivilegeModify                 CatalogGrantPrivilege = "MODIFY"
	CatalogGrantPrivilegeReadVolume             CatalogGrantPrivilege = "READ_VOLUME"
	CatalogGrantPrivilegeRefresh                CatalogGrantPrivilege = "REFRESH"
	CatalogGrantPrivilegeSelect                 CatalogGrantPrivilege = "SELECT"
	CatalogGrantPrivilegeUseCatalog             CatalogGrantPrivilege = "USE_CATALOG"
	CatalogGrantPrivilegeUseConnection          CatalogGrantPrivilege = "USE_CONNECTION"
	CatalogGrantPrivilegeUseSchema              CatalogGrantPrivilege = "USE_SCHEMA"
	CatalogGrantPrivilegeWriteVolume            CatalogGrantPrivilege = "WRITE_VOLUME"
)

// Values returns all valid CatalogGrantPrivilege values
func (CatalogGrantPrivilege) Values() []CatalogGrantPrivilege {
	return []CatalogGrantPrivilege{
		CatalogGrantPrivilegeAllPrivileges,
		CatalogGrantPrivilegeApplyTag,
		CatalogGrantPrivilegeCreateConnection,
		CatalogGrantPrivilegeCreateExternalLocation,
		CatalogGrantPrivilegeCreateExternalTable,
		CatalogGrantPrivilegeCreateExternalVolume,
		CatalogGrantPrivilegeCreateForeignCatalog,
		CatalogGrantPrivilegeCreateFunction,
		CatalogGrantPrivilegeCreateManagedStorage,
		CatalogGrantPrivilegeCreateMaterializedView,
		CatalogGrantPrivilegeCreateModel,
		CatalogGrantPrivilegeCreateSchema,
		CatalogGrantPrivilegeCreateStorageCredential,
		CatalogGrantPrivilegeCreateTable,
		CatalogGrantPrivilegeCreateVolume,
		CatalogGrantPrivilegeExecute,
		CatalogGrantPrivilegeManage,
		CatalogGrantPrivilegeModify,
		CatalogGrantPrivilegeReadVolume,
		CatalogGrantPrivilegeRefresh,
		CatalogGrantPrivilegeSelect,
		CatalogGrantPrivilegeUseCatalog,
		CatalogGrantPrivilegeUseConnection,
		CatalogGrantPrivilegeUseSchema,
		CatalogGrantPrivilegeWriteVolume,
	}
}

// CatalogGrant holds the grant level settings for a single principal in Unity Catalog.
// Multiple of these can be defined on any catalog.
type CatalogGrant struct {
	Privileges []CatalogGrantPrivilege `json:"privileges"`

	Principal string `json:"principal"`
}

type Catalog struct {
	BaseResource
	catalog.CreateCatalog

	// Whether predictive optimization should be enabled for this object and objects under it.
	// This field is only used for updates and cannot be set during creation.
	EnablePredictiveOptimization catalog.EnablePredictiveOptimization `json:"enable_predictive_optimization,omitempty"`

	// Whether the current securable is accessible from all workspaces or a specific set of workspaces.
	// This field is only used for updates and cannot be set during creation.
	IsolationMode catalog.CatalogIsolationMode `json:"isolation_mode,omitempty"`

	// List of grants to apply on this catalog.
	Grants []CatalogGrant `json:"grants,omitempty"`
}

func (c *Catalog) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	log.Tracef(ctx, "Checking if catalog with name=%s exists", name)

	_, err := w.Catalogs.GetByName(ctx, name)
	if err != nil {
		log.Debugf(ctx, "catalog with name %s does not exist: %v", name, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*Catalog) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "catalog",
		PluralName:    "catalogs",
		SingularTitle: "Catalog",
		PluralTitle:   "Catalogs",
	}
}

func (c *Catalog) InitializeURL(baseURL url.URL) {
	if c.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + strings.ReplaceAll(c.ID, ".", "/")
	c.URL = baseURL.String()
}

func (c *Catalog) GetURL() string {
	return c.URL
}

func (c *Catalog) GetName() string {
	return c.Name
}

func (c *Catalog) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c Catalog) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}
