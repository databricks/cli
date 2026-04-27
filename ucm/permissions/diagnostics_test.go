package permissions_test

import (
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/cli/ucm/permissions"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testUser = "alice@example.com"

func userFixture(cats map[string]*resources.Catalog, schemas map[string]*resources.Schema) *ucm.Ucm {
	u := &ucm.Ucm{}
	u.Config.Resources.Catalogs = cats
	u.Config.Resources.Schemas = schemas
	u.CurrentUser = &config.User{User: &iam.User{UserName: testUser}}
	return u
}

func manageAssignment() []catalog.EffectivePrivilegeAssignment {
	return []catalog.EffectivePrivilegeAssignment{{
		Principal:  testUser,
		Privileges: []catalog.EffectivePrivilege{{Privilege: catalog.PrivilegeManage}},
	}}
}

func readOnlyAssignment() []catalog.EffectivePrivilegeAssignment {
	return []catalog.EffectivePrivilegeAssignment{{
		Principal:  testUser,
		Privileges: []catalog.EffectivePrivilege{{Privilege: catalog.PrivilegeUseCatalog}},
	}}
}

func TestPermissionDiagnosticsEmptyWhenAllPass(t *testing.T) {
	u := userFixture(
		map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}}},
		map[string]*resources.Schema{"bronze": {CreateSchema: catalog.CreateSchema{Name: "bronze", CatalogName: "main"}}},
	)
	ws := mocks.NewMockWorkspaceClient(t)
	ws.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, catalog.GetEffectiveRequest{
			SecurableType: string(catalog.SecurableTypeCatalog),
			FullName:      "main",
			Principal:     testUser,
		}).
		Return(&catalog.EffectivePermissionsList{PrivilegeAssignments: manageAssignment()}, nil)
	ws.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, catalog.GetEffectiveRequest{
			SecurableType: string(catalog.SecurableTypeSchema),
			FullName:      "main.bronze",
			Principal:     testUser,
		}).
		Return(&catalog.EffectivePermissionsList{PrivilegeAssignments: manageAssignment()}, nil)
	u.SetWorkspaceClient(ws.WorkspaceClient)

	diags := permissions.PermissionDiagnostics(t.Context(), u)
	assert.Empty(t, diags)
}

func TestPermissionDiagnosticsEmitsWhenUserLacksManage(t *testing.T) {
	u := userFixture(
		map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}}},
		map[string]*resources.Schema{"bronze": {CreateSchema: catalog.CreateSchema{Name: "bronze", CatalogName: "main"}}},
	)
	ws := mocks.NewMockWorkspaceClient(t)
	ws.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, mock.MatchedBy(func(r catalog.GetEffectiveRequest) bool {
			return r.SecurableType == string(catalog.SecurableTypeCatalog)
		})).
		Return(&catalog.EffectivePermissionsList{PrivilegeAssignments: readOnlyAssignment()}, nil)
	ws.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, mock.MatchedBy(func(r catalog.GetEffectiveRequest) bool {
			return r.SecurableType == string(catalog.SecurableTypeSchema)
		})).
		Return(&catalog.EffectivePermissionsList{PrivilegeAssignments: readOnlyAssignment()}, nil)
	u.SetWorkspaceClient(ws.WorkspaceClient)

	diags := permissions.PermissionDiagnostics(t.Context(), u)
	require.Len(t, diags, 2)
	for _, d := range diags {
		assert.Equal(t, diag.Error, d.Severity)
		assert.Contains(t, d.Summary, testUser)
		assert.Contains(t, d.Summary, "MANAGE")
	}
}

func TestPermissionDiagnosticsSkipsWhenCatalogNotFound(t *testing.T) {
	u := userFixture(
		map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}}},
		nil,
	)
	ws := mocks.NewMockWorkspaceClient(t)
	ws.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, mock.Anything).
		Return(nil, &apierr.APIError{StatusCode: http.StatusNotFound, ErrorCode: "CATALOG_DOES_NOT_EXIST"})
	u.SetWorkspaceClient(ws.WorkspaceClient)

	diags := permissions.PermissionDiagnostics(t.Context(), u)
	assert.Empty(t, diags)
}

func TestPermissionDiagnosticsEmitsWhenForbidden(t *testing.T) {
	u := userFixture(
		map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}}},
		nil,
	)
	ws := mocks.NewMockWorkspaceClient(t)
	ws.GetMockGrantsAPI().EXPECT().
		GetEffective(mock.Anything, mock.Anything).
		Return(nil, &apierr.APIError{StatusCode: http.StatusForbidden, ErrorCode: "PERMISSION_DENIED"})
	u.SetWorkspaceClient(ws.WorkspaceClient)

	diags := permissions.PermissionDiagnostics(t.Context(), u)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
}

func TestPermissionDiagnosticsNoopWhenNoResources(t *testing.T) {
	u := userFixture(nil, nil)
	ws := mocks.NewMockWorkspaceClient(t)
	u.SetWorkspaceClient(ws.WorkspaceClient)

	diags := permissions.PermissionDiagnostics(t.Context(), u)
	assert.Empty(t, diags)
}

func TestPermissionDiagnosticsSkipsWhenNoCurrentUser(t *testing.T) {
	u := &ucm.Ucm{}
	u.Config.Resources.Catalogs = map[string]*resources.Catalog{"main": {CreateCatalog: catalog.CreateCatalog{Name: "main"}}}
	ws := mocks.NewMockWorkspaceClient(t)
	u.SetWorkspaceClient(ws.WorkspaceClient)

	diags := permissions.PermissionDiagnostics(t.Context(), u)
	assert.Empty(t, diags)
}
