package resources

import (
	"context"
	"net/url"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"

	"github.com/databricks/cli/libs/log"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type SchemaGrantPrivilege string

const (
	SchemaGrantPrivilegeAllPrivileges  SchemaGrantPrivilege = "ALL_PRIVILEGES"
	SchemaGrantPrivilegeApplyTag       SchemaGrantPrivilege = "APPLY_TAG"
	SchemaGrantPrivilegeCreateFunction SchemaGrantPrivilege = "CREATE_FUNCTION"
	SchemaGrantPrivilegeCreateTable    SchemaGrantPrivilege = "CREATE_TABLE"
	SchemaGrantPrivilegeCreateVolume   SchemaGrantPrivilege = "CREATE_VOLUME"
	SchemaGrantPrivilegeManage         SchemaGrantPrivilege = "MANAGE"
	SchemaGrantPrivilegeUseSchema      SchemaGrantPrivilege = "USE_SCHEMA"
	SchemaGrantPrivilegeExecute        SchemaGrantPrivilege = "EXECUTE"
	SchemaGrantPrivilegeModify         SchemaGrantPrivilege = "MODIFY"
	SchemaGrantPrivilegeRefresh        SchemaGrantPrivilege = "REFRESH"
	SchemaGrantPrivilegeSelect         SchemaGrantPrivilege = "SELECT"
	SchemaGrantPrivilegeReadVolume     SchemaGrantPrivilege = "READ_VOLUME"
	SchemaGrantPrivilegeWriteVolume    SchemaGrantPrivilege = "WRITE_VOLUME"
)

// Values returns all valid SchemaGrantPrivilege values
func (SchemaGrantPrivilege) Values() []SchemaGrantPrivilege {
	return []SchemaGrantPrivilege{
		SchemaGrantPrivilegeAllPrivileges,
		SchemaGrantPrivilegeApplyTag,
		SchemaGrantPrivilegeCreateFunction,
		SchemaGrantPrivilegeCreateTable,
		SchemaGrantPrivilegeCreateVolume,
		SchemaGrantPrivilegeManage,
		SchemaGrantPrivilegeUseSchema,
		SchemaGrantPrivilegeExecute,
		SchemaGrantPrivilegeModify,
		SchemaGrantPrivilegeRefresh,
		SchemaGrantPrivilegeSelect,
		SchemaGrantPrivilegeReadVolume,
		SchemaGrantPrivilegeWriteVolume,
	}
}

// SchemaGrant holds the grant level settings for a single principal in Unity Catalog.
// Multiple of these can be defined on any schema.
type SchemaGrant struct {
	Privileges []SchemaGrantPrivilege `json:"privileges"`

	Principal string `json:"principal"`
}

type Schema struct {
	BaseResource
	catalog.CreateSchema
	// List of grants to apply on this schema.
	Grants []SchemaGrant `json:"grants,omitempty"`
}

func (s *Schema) Exists(ctx context.Context, w *databricks.WorkspaceClient, fullName string) (bool, error) {
	log.Tracef(ctx, "Checking if schema with fullName=%s exists", fullName)

	_, err := w.Schemas.GetByFullName(ctx, fullName)
	if err != nil {
		log.Debugf(ctx, "schema with full name %s does not exist: %v", fullName, err)

		if apierr.IsMissing(err) {
			return false, nil
		}

		return false, err
	}
	return true, nil
}

func (*Schema) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "schema",
		PluralName:    "schemas",
		SingularTitle: "Schema",
		PluralTitle:   "Schemas",
	}
}

func (s *Schema) InitializeURL(baseURL url.URL) {
	if s.ID == "" {
		return
	}
	baseURL.Path = "explore/data/" + strings.ReplaceAll(s.ID, ".", "/")
	s.URL = baseURL.String()
}

func (s *Schema) GetURL() string {
	return s.URL
}

func (s *Schema) GetName() string {
	return s.Name
}

func (s *Schema) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s Schema) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}
